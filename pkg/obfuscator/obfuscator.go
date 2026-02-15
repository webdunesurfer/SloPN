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
	
	// sessionMap tracks which remote addresses are authenticated
	// remoteAddr -> lastSeen
	sessions    map[string]time.Time
	sessionsMu  sync.RWMutex

	// proxySessions tracks unauthorized probes for mirroring
	// clientAddr -> proxy session
	proxySessions map[string]*proxySession
	proxyMu       sync.RWMutex
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
		sessions:   make(map[string]time.Time),
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
		for addr, lastSeen := range c.sessions {
			if now.Sub(lastSeen) > SessionTimeout {
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
		
		// 1. Check if already authenticated
		c.sessionsMu.RLock()
		_, active := c.sessions[remoteKey]
		c.sessionsMu.RUnlock()

		if active {
			c.sessionsMu.Lock()
			c.sessions[remoteKey] = time.Now()
			c.sessionsMu.Unlock()

			// Unmask the payload
			// In active mode, the first MagicHeaderLen bytes were only in the FIRST packet.
			// Subsequent packets are just XORed data.
			c.xor(buf[:n], []byte(remoteKey))
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
			fmt.Printf("[DEBUG] Reality AUTH SUCCESS from %v\n", addr)
			// SUCCESS: Authenticate this IP
			c.sessionsMu.Lock()
			c.sessions[remoteKey] = time.Now()
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
	fmt.Printf("[DEBUG] Reality handleMirror: %d bytes from %v -> mirroring to %v\n", len(data), addr, c.mimicAddr)
	if c.mimicAddr == nil {
		return // Silent drop
	}

	remoteKey := addr.String()
	
	c.proxyMu.Lock()
	sess, exists := c.proxySessions[remoteKey]
	if !exists {
		// Create new proxy connection
		conn, err := net.DialUDP("udp", nil, c.mimicAddr)
		if err != nil {
			c.proxyMu.Unlock()
			return
		}

		sess = &proxySession{
			conn:       conn,
			lastActive: time.Now(),
		}
		c.proxySessions[remoteKey] = sess
		
		// Start response listener for this proxy session
		go func(clientAddr net.Addr, proxyConn *net.UDPConn) {
			buf := make([]byte, 2048)
			for {
				proxyConn.SetReadDeadline(time.Now().Add(ProxyTimeout))
				n, err := proxyConn.Read(buf)
				if err != nil {
					return // End of session
				}

				// Forward response back to the original client
				c.PacketConn.WriteTo(buf[:n], clientAddr)
				
				// Update activity
				c.proxyMu.Lock()
				if s, ok := c.proxySessions[remoteKey]; ok {
					s.lastActive = time.Now()
				}
				c.proxyMu.Unlock()
			}
		}(addr, conn)
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
	_, active := c.sessions[remoteKey]
	c.sessionsMu.RUnlock()

	if active {
		// Just XOR and send
		fmt.Printf("[DEBUG] Reality WriteTo: Sending XORed packet (%d bytes) to %v\n", len(p), addr)
		copy(buf, p)
		c.xor(buf[:len(p)], []byte(remoteKey))
		return c.PacketConn.WriteTo(buf[:len(p)], addr)
	}

	// First packet: Add Magic Header
	fmt.Printf("[DEBUG] Reality WriteTo: Sending FIRST packet with Magic Header. Payload: %d bytes, Total: %d bytes\n", len(p), MagicHeaderLen+len(p))
	salt := make([]byte, 8)
	rand.Read(salt)

	mac := hmac.New(sha256.New, c.authKey)
	mac.Write(salt)
	signature := mac.Sum(nil)[:24]

	copy(buf[0:8], salt)
	copy(buf[8:32], signature)
	
	// XOR the payload
	payload := buf[MagicHeaderLen : MagicHeaderLen+len(p)]
	copy(payload, p)
	c.xor(payload, salt)

	// Update session so we don't send the header again
	c.sessionsMu.Lock()
	c.sessions[remoteKey] = time.Now()
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
