package obfuscator

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/hkdf"
)

// RealityConn implements a "Reality-style" stealth transport.
// It mimics a legitimate UDP service (like DTLS or high-entropy QUIC)
// and redirects unauthorized probes to a "mimic" target.
type RealityConn struct {
	net.PacketConn
	key         []byte
	authKey     []byte // Used for the "Magic Packet" signature
	mimicAddr   *net.UDPAddr
	pool        *sync.Pool
	
	// sessions tracks authenticated VPN flows
	// remoteAddr -> sessionState
	sessions    map[string]*sessionState
	sessionsMu  sync.RWMutex

	// proxySessions tracks unauthorized probes for mirroring
	// clientAddr -> proxy session
	proxySessions map[string]*proxySession
	proxyMu       sync.RWMutex
}

type sessionState struct {
	salt       []byte
	lastActive time.Time
}

type proxySession struct {
	conn       *net.UDPConn
	lastActive time.Time
}

const (
	// MagicHeaderLen is the size of our "secret handshake"
	// 8 bytes of random-looking salt + 24 bytes of HMAC-SHA256
	MagicHeaderLen = 32
	SessionTimeout = 5 * time.Minute
	ProxyTimeout   = 2 * time.Minute
)

// NewRealityConn creates a new Reality-style connection wrapper.
// mimicTarget is the address (IP:Port) of a real server to mirror if auth fails.
func NewRealityConn(conn net.PacketConn, secret string, mimicTarget string) *RealityConn {
	// Derive keys using HKDF
	hash := sha256.New
	kdf := hkdf.New(hash, []byte(secret), nil, []byte("slopn-reality-v1"))
	
	key := make([]byte, 32)     // XOR Key
	authKey := make([]byte, 32) // HMAC Key
	io.ReadFull(kdf, key)
	io.ReadFull(kdf, authKey)

	var mAddr *net.UDPAddr
	if mimicTarget != "" {
		mAddr, _ = net.ResolveUDPAddr("udp", mimicTarget)
	}

	rc := &RealityConn{
		PacketConn: conn,
		key:        key,
		authKey:    authKey,
		mimicAddr:  mAddr,
		sessions:   make(map[string]*sessionState),
		proxySessions: make(map[string]*proxySession),
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 2048)
			},
		},
	}

	// Cleanup stale sessions
	go rc.cleanupLoop()

	return rc
}

func (c *RealityConn) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		// Cleanup VPN sessions
		c.sessionsMu.Lock()
		now := time.Now()
		for addr, sess := range c.sessions {
			if now.Sub(sess.lastActive) > SessionTimeout {
				delete(c.sessions, addr)
			}
		}
		c.sessionsMu.Unlock()

		// Cleanup proxy sessions
		c.proxyMu.Lock()
		for addr, sess := range c.proxySessions {
			if now.Sub(sess.lastActive) > ProxyTimeout {
				sess.conn.Close()
				delete(c.proxySessions, addr)
			}
		}
		c.proxyMu.Unlock()
	}
}

// xor applies the rolling XOR mask
func (c *RealityConn) xor(p []byte, seed []byte) {
	kLen := len(c.key)
	// Mix in the seed (salt) from the packet to ensure unique patterns
	var salt uint32
	if len(seed) >= 4 {
		salt = binary.BigEndian.Uint32(seed[:4])
	}
	
	offset := int(salt) % kLen
	for i := 0; i < len(p); i++ {
		p[i] ^= c.key[(i+offset)%kLen]
	}
}

// ReadFrom performs the "Gatekeeper" logic
func (c *RealityConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	buf := c.pool.Get().([]byte)
	defer c.pool.Put(buf)

	for {
		n, addr, err = c.PacketConn.ReadFrom(buf)
		if err != nil {
			return 0, addr, err
		}

		remoteKey := addr.String()
		fmt.Printf("[DEBUG] Reality Gatekeeper: Received %d bytes from %v\n", n, addr)
		
		// 1. Check if already authenticated
		c.sessionsMu.RLock()
		sess, active := c.sessions[remoteKey]
		c.sessionsMu.RUnlock()

		if active {
			c.sessionsMu.Lock()
			sess.lastActive = time.Now()
			c.sessionsMu.Unlock()

			// Unmask the payload using the session's salt
			c.xor(buf[:n], sess.salt)
			copy(p, buf[:n])
			return n, addr, nil
		}

		// 2. New Flow: Must contain Magic Header
		if n < MagicHeaderLen {
			c.handleMirror(buf[:n], addr)
			continue
		}

		// Header = Salt (8b) + Signature (24b)
		salt := buf[:8]
		signature := buf[8:32]
		
		// Verify HMAC
		mac := hmac.New(sha256.New, c.authKey)
		mac.Write(salt)
		expected := mac.Sum(nil)[:24]

		if hmac.Equal(signature, expected) {
			// SUCCESS: Authenticate this IP and store the salt
			s := &sessionState{
				salt:       make([]byte, 8),
				lastActive: time.Now(),
			}
			copy(s.salt, salt)

			c.sessionsMu.Lock()
			c.sessions[remoteKey] = s
			c.sessionsMu.Unlock()

			// The payload starts after the header
			payload := buf[MagicHeaderLen:n]
			c.xor(payload, salt)
			
			copy(p, payload)
			return n - MagicHeaderLen, addr, nil
		}

		// 3. FAILED: Act as a mirror
		c.handleMirror(buf[:n], addr)
	}
}

func (c *RealityConn) handleMirror(data []byte, addr net.Addr) {
	if c.mimicAddr == nil {
		return // Silent drop
	}

	remoteKey := addr.String()
	
	c.proxyMu.Lock()
	sess, exists := c.proxySessions[remoteKey]
	if !exists {
		fmt.Printf("[DEBUG] Creating new proxy session for %v -> %v\n", addr, c.mimicAddr)
		// Create new proxy connection
		conn, err := net.DialUDP("udp", nil, c.mimicAddr)
		if err != nil {
			fmt.Printf("[DEBUG] Failed to dial mimic target: %v\n", err)
			c.proxyMu.Unlock()
			return
		}

		sess = &proxySession{
			conn:       conn,
			lastActive: time.Now(),
		}
		c.proxySessions[remoteKey] = sess
		
		// Start response listener for this proxy session
		go func(clientAddr net.Addr, proxyConn *net.UDPConn, key string) {
			buf := make([]byte, 2048)
			for {
				proxyConn.SetReadDeadline(time.Now().Add(ProxyTimeout))
				n, err := proxyConn.Read(buf)
				if err != nil {
					fmt.Printf("[DEBUG] Proxy session closed for %v: %v\n", clientAddr, err)
					return // End of session
				}

				fmt.Printf("[DEBUG] Proxy forwarding %d bytes back to %v\n", n, clientAddr)
				// Forward response back to the original client
				c.PacketConn.WriteTo(buf[:n], clientAddr)
				
				// Update activity
				c.proxyMu.Lock()
				if s, ok := c.proxySessions[key]; ok {
					s.lastActive = time.Now()
				}
				c.proxyMu.Unlock()
			}
		}(addr, conn, remoteKey)
	}
	sess.lastActive = time.Now()
	c.proxyMu.Unlock()

	// Forward the probe data to the mimic target
	sess.conn.Write(data)
}

// WriteTo adds the Magic Header for the first packet of a session
func (c *RealityConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	buf := c.pool.Get().([]byte)
	defer c.pool.Put(buf)

	remoteKey := addr.String()
	c.sessionsMu.RLock()
	sess, active := c.sessions[remoteKey]
	c.sessionsMu.RUnlock()

	if active {
		// Just XOR and send using the established session salt
		copy(buf, p)
		c.xor(buf[:len(p)], sess.salt)
		return c.PacketConn.WriteTo(buf[:len(p)], addr)
	}

	// First packet: Add Magic Header
	salt := make([]byte, 8)
	rand.Read(salt)

	mac := hmac.New(sha256.New, c.authKey)
	mac.Write(salt)
	signature := mac.Sum(nil)[:24]

	copy(buf[0:8], salt)
	copy(buf[8:32], signature)
	
	// XOR the payload using this new salt
	payload := buf[MagicHeaderLen : MagicHeaderLen+len(p)]
	copy(payload, p)
	c.xor(payload, salt)

	// NOTE: We do NOT mark the session active in WriteTo yet.
	// We only transition to XOR-only mode AFTER we receive an authenticated packet 
	// from the peer in ReadFrom (server side) or when the client knows it started 
	// the session. 
	
	// CRITICAL FIX: The client needs to transition to 'active' too, but only after 
	// it's sure the server has received the Magic Header. To keep it simple and 
	// robust against packet loss, we'll mark it active here but we fix the XOR seed 
	// to be the SALT instead of the remoteAddr.

	s := &sessionState{
		salt:       make([]byte, 8),
		lastActive: time.Now(),
	}
	copy(s.salt, salt)
	c.sessionsMu.Lock()
	c.sessions[remoteKey] = s
	c.sessionsMu.Unlock()

	_, err = c.PacketConn.WriteTo(buf[:MagicHeaderLen+len(p)], addr)
	return len(p), err
}

func (c *RealityConn) LocalAddr() net.Addr                { return c.PacketConn.LocalAddr() }
func (c *RealityConn) SetDeadline(t time.Time) error      { return c.PacketConn.SetDeadline(t) }
func (c *RealityConn) SetReadDeadline(t time.Time) error  { return c.PacketConn.SetReadDeadline(t) }
func (c *RealityConn) SetWriteDeadline(t time.Time) error { return c.PacketConn.SetWriteDeadline(t) }
func (c *RealityConn) Close() error                       { return c.PacketConn.Close() }

func (c *RealityConn) SetReadBuffer(bytes int) error {
	if u, ok := c.PacketConn.(interface{ SetReadBuffer(int) error }); ok {
		return u.SetReadBuffer(bytes)
	}
	return nil
}

func (c *RealityConn) SetWriteBuffer(bytes int) error {
	if u, ok := c.PacketConn.(interface{ SetWriteBuffer(int) error }); ok {
		return u.SetWriteBuffer(bytes)
	}
	return nil
}
