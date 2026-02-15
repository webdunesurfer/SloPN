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
type RealityConn struct {
	net.PacketConn
	key         []byte
	authKey     []byte
	mimicAddr   *net.UDPAddr
	pool        *sync.Pool
	
	sessions    map[string]*sessionState
	sessionsMu  sync.RWMutex

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
	MagicHeaderLen = 32
	SessionTimeout = 5 * time.Minute
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
		sessions:   make(map[string]*sessionState),
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
		c.sessionsMu.Lock()
		now := time.Now()
		for addr, sess := range c.sessions {
			if now.Sub(sess.lastActive) > SessionTimeout {
				delete(c.sessions, addr)
			}
		}
		c.sessionsMu.Unlock()

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

func (c *RealityConn) xor(p []byte, seed []byte) {
	kLen := len(c.key)
	var salt uint32
	if len(seed) >= 4 {
		salt = binary.BigEndian.Uint32(seed[:4])
	}
	offset := int(salt) % kLen
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
		
		c.sessionsMu.RLock()
		sess, active := c.sessions[remoteKey]
		c.sessionsMu.RUnlock()

		if active {
			c.sessionsMu.Lock()
			sess.lastActive = time.Now()
			c.sessionsMu.Unlock()
			c.xor(buf[:n], sess.salt)
			copy(p, buf[:n])
			return n, addr, nil
		}

		if n < MagicHeaderLen {
			if n > 0 && n < 64 {
				c.PacketConn.WriteTo(buf[:n], addr)
				continue
			}
			c.handleMirror(buf[:n], addr)
			continue
		}

		salt := buf[:8]
		signature := buf[8:32]
		mac := hmac.New(sha256.New, c.authKey)
		mac.Write(salt)
		expected := mac.Sum(nil)[:24]

		if hmac.Equal(signature, expected) {
			s := &sessionState{
				salt:       make([]byte, 8),
				lastActive: time.Now(),
			}
			copy(s.salt, salt)
			c.sessionsMu.Lock()
			c.sessions[remoteKey] = s
			c.sessionsMu.Unlock()
			payload := buf[MagicHeaderLen:n]
			c.xor(payload, salt)
			copy(p, payload)
			return n - MagicHeaderLen, addr, nil
		}

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

	remoteKey := addr.String()
	c.sessionsMu.RLock()
	sess, active := c.sessions[remoteKey]
	c.sessionsMu.RUnlock()

	if active {
		copy(buf, p)
		c.xor(buf[:len(p)], sess.salt)
		return c.PacketConn.WriteTo(buf[:len(p)], addr)
	}

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
