package obfuscator

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/hkdf"
)

// RealityConn implements a "Reality-style" stealth transport.
// v0.9.6: Uses First-Packet-Obfuscation (FPO) to transition from masked handshakes
// to clean QUIC flows after authorization.
type RealityConn struct {
	net.PacketConn
	key         []byte
	authKey     []byte
	mimicAddr   *net.UDPAddr
	pool        *sync.Pool
	
	// proxySessions tracks unauthorized probes for mirroring
	proxySessions map[string]*proxySession
	proxyMu       sync.RWMutex

	// FPO (First-Packet-Obfuscation) state
	authIPs     map[string]time.Time
	handshakeMu sync.RWMutex
	sentCount   map[string]int
}

type proxySession struct {
	conn       *net.UDPConn
	lastActive time.Time
}

const (
	// MagicHeaderLen is Salt (8b) + HMAC-SHA256 (24b)
	MagicHeaderLen = 32
	ProxyTimeout   = 2 * time.Minute
	AuthTimeout    = 1 * time.Hour
	HandshakeLimit = 20 // Number of packets to obfuscate before switching to clean mode
)

func NewRealityConn(conn net.PacketConn, secret string, mimicTarget string) *RealityConn {
	hash := sha256.New
	kdf := hkdf.New(hash, []byte(secret), nil, []byte("slopn-reality-v1"))
	
	key := make([]byte, 32)
	authKey := make([]byte, 32)
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
		proxySessions: make(map[string]*proxySession),
		authIPs:       make(map[string]time.Time),
		sentCount:     make(map[string]int),
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 2048)
			},
		},
	}

	go rc.cleanupLoop()
	return rc
}

func (c *RealityConn) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		c.proxyMu.Lock()
		for addr, sess := range c.proxySessions {
			if time.Since(sess.lastActive) > ProxyTimeout {
				sess.conn.Close()
				delete(c.proxySessions, addr)
			}
		}
		c.proxyMu.Unlock()

		c.handshakeMu.Lock()
		for addr, t := range c.authIPs {
			if time.Since(t) > AuthTimeout {
				delete(c.authIPs, addr)
				delete(c.sentCount, addr)
			}
		}
		c.handshakeMu.Unlock()
	}
}

func (c *RealityConn) xor(p []byte, salt []byte) {
	kLen := len(c.key)
	var s uint32
	if len(salt) >= 4 {
		s = binary.BigEndian.Uint32(salt[:4])
	}
	offset := int(s) % kLen
	for i := 0; i < len(p); i++ {
		p[i] ^= c.key[(i+offset)%kLen]
	}
}

func (c *RealityConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	buf := c.pool.Get().([]byte)
	defer c.pool.Put(buf)

	for {
		n, addr, err = c.PacketConn.ReadFrom(buf)
		if err != nil {
			return 0, addr, err
		}

		remoteKey := addr.String()
		ip, _, _ := net.SplitHostPort(remoteKey)

		// 1. Try to parse as FPO (First-Packet-Obfuscation)
		// Even if whitelisted, we check for FPO first to handle the overlap period
		if n >= MagicHeaderLen {
			salt := buf[:8]
			signature := buf[8:32]
			mac := hmac.New(sha256.New, c.authKey)
			mac.Write(salt)
			expected := mac.Sum(nil)[:24]

			if hmac.Equal(signature, expected) {
				padLen := int(salt[0] & 31)
				realPayloadLen := n - MagicHeaderLen - padLen
				if realPayloadLen >= 0 {
					// Promotion: Ensure IP is whitelisted
					c.handshakeMu.Lock()
					if _, exists := c.authIPs[ip]; !exists {
						c.authIPs[ip] = time.Now()
					}
					c.handshakeMu.Unlock()

					payload := buf[MagicHeaderLen : MagicHeaderLen+realPayloadLen]
					c.xor(payload, salt)
					copy(p, payload)
					return realPayloadLen, addr, nil
				}
			}
		}

		// 2. Fallback Path: Clean QUIC (only if whitelisted)
		c.handshakeMu.RLock()
		_, whitelisted := c.authIPs[ip]
		c.handshakeMu.RUnlock()

		if whitelisted {
			copy(p, buf[:n])
			return n, addr, nil
		}

		// 3. Unauthorized probe: Mirror to target
		c.handleMirror(buf[:n], addr)
	}
}

func (c *RealityConn) handleMirror(data []byte, addr net.Addr) {
	if c.mimicAddr == nil {
		return
	}

	remoteKey := addr.String()
	c.proxyMu.Lock()
	sess, exists := c.proxySessions[remoteKey]
	if !exists {
		conn, err := net.DialUDP("udp", nil, c.mimicAddr)
		if err != nil {
			c.proxyMu.Unlock()
			return
		}
		sess = &proxySession{conn: conn, lastActive: time.Now()}
		c.proxySessions[remoteKey] = sess
		
		go func(clientAddr net.Addr, proxyConn *net.UDPConn, key string) {
			buf := make([]byte, 2048)
			for {
				proxyConn.SetReadDeadline(time.Now().Add(ProxyTimeout))
				n, err := proxyConn.Read(buf)
				if err != nil {
					return
				}
				c.PacketConn.WriteTo(buf[:n], clientAddr)
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
	
	sess.conn.Write(data)
}

func (c *RealityConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	remoteKey := addr.String()

	// 1. Check if we should use Clean Mode
	c.handshakeMu.Lock()
	count := c.sentCount[remoteKey]
	if count >= HandshakeLimit {
		c.handshakeMu.Unlock()
		return c.PacketConn.WriteTo(p, addr)
	}
	c.sentCount[remoteKey]++
	c.handshakeMu.Unlock()

	// 2. FPO Mode: Obfuscate packet
	buf := c.pool.Get().([]byte)
	defer c.pool.Put(buf)

	// Prepend Header: Salt(8) + HMAC(24)
	salt := make([]byte, 8)
	rand.Read(salt)
	
	mac := hmac.New(sha256.New, c.authKey)
	mac.Write(salt)
	signature := mac.Sum(nil)[:24]

	copy(buf[0:8], salt)
	copy(buf[8:32], signature)
	
	payload := buf[MagicHeaderLen : MagicHeaderLen+len(p)]
	copy(payload, p)
	c.xor(payload, salt)

	// Add random padding (0-31 bytes) to break packet size signatures
	padLen := int(salt[0] & 31)
	totalLen := MagicHeaderLen + len(p) + padLen
	if totalLen > len(buf) {
		padLen = 0 // Safety check
		totalLen = MagicHeaderLen + len(p)
	} else {
		rand.Read(buf[MagicHeaderLen+len(p) : totalLen])
	}

	_, err = c.PacketConn.WriteTo(buf[:totalLen], addr)
	return len(p), err
}

func (c *RealityConn) LocalAddr() net.Addr                { return c.PacketConn.LocalAddr() }
func (c *RealityConn) SetDeadline(t time.Time) error      { return c.PacketConn.SetDeadline(t) }
func (c *RealityConn) SetReadDeadline(t time.Time) error  { return c.PacketConn.SetReadDeadline(t) }
func (c *RealityConn) SetWriteDeadline(t time.Time) error { return c.PacketConn.SetWriteDeadline(t) }
func (c *RealityConn) Close() error                       { return c.PacketConn.Close() }

// Buffer optimizations for quic-go
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
