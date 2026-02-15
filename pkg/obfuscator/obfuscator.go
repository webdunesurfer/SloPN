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
// In v0.9.2+, this is a stateless authenticated wrapper where EVERY packet
// includes a Magic Header for maximum robustness against UDP loss and DPI.
type RealityConn struct {
	net.PacketConn
	key         []byte
	authKey     []byte
	mimicAddr   *net.UDPAddr
	pool        *sync.Pool
	
	// proxySessions tracks unauthorized probes for mirroring
	proxySessions map[string]*proxySession
	proxyMu       sync.RWMutex
}

type proxySession struct {
	conn       *net.UDPConn
	lastActive time.Time
}

const (
	// MagicHeaderLen is Salt (8b) + HMAC-SHA256 (24b)
	MagicHeaderLen = 32
	ProxyTimeout   = 2 * time.Minute
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

		if n < MagicHeaderLen {
			c.handleMirror(buf[:n], addr)
			continue
		}

		salt := buf[:8]
		signature := buf[8:32]
		mac := hmac.New(sha256.New, c.authKey)
		mac.Write(salt)
		expected := mac.Sum(nil)[:24]

		if hmac.Equal(signature, expected) {
			// Authorized SloPN packet
			payload := buf[MagicHeaderLen:n]
			c.xor(payload, salt)
			copy(p, payload)
			return n - MagicHeaderLen, addr, nil
		}

		// Unauthorized probe
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

	_, err = c.PacketConn.WriteTo(buf[:MagicHeaderLen+len(p)], addr)
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
