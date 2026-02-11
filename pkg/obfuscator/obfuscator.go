package obfuscator

import (
	"crypto/sha256"
	"net"
	"sync"
)

// ObfuscatedConn wraps a net.PacketConn and XORs all packets
type ObfuscatedConn struct {
	net.PacketConn
	key []byte
	pool *sync.Pool
}

// NewObfuscatedConn creates a new XOR-masked connection
func NewObfuscatedConn(conn net.PacketConn, secret string) *ObfuscatedConn {
	h := sha256.Sum256([]byte(secret))
	return &ObfuscatedConn{
		PacketConn: conn,
		key:        h[:],
		pool: &sync.Pool{
			New: func() interface{} {
				// Allocate buffers large enough for MTU + overhead
				return make([]byte, 2048)
			},
		},
	}
}

func (c *ObfuscatedConn) xor(p []byte) {
	keyLen := len(c.key)
	for i := 0; i < len(p); i++ {
		p[i] ^= c.key[i%keyLen]
	}
}

// ReadFrom unmasks incoming packets
func (c *ObfuscatedConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, addr, err = c.PacketConn.ReadFrom(p)
	if err == nil {
		c.xor(p[:n])
	}
	return
}

// WriteTo masks outgoing packets using a pooled buffer
func (c *ObfuscatedConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	// Use pooled buffer to avoid allocation per packet
	buf := c.pool.Get().([]byte)
	defer c.pool.Put(buf)

	if len(p) > len(buf) {
		// Fallback for oversized packets (should not happen with MTU 1200)
		tmp := make([]byte, len(p))
		copy(tmp, p)
		c.xor(tmp)
		return c.PacketConn.WriteTo(tmp, addr)
	}

	copy(buf, p)
	c.xor(buf[:len(p)])
	return c.PacketConn.WriteTo(buf[:len(p)], addr)
}

// SetReadBuffer satisfies quic-go performance optimizations
func (c *ObfuscatedConn) SetReadBuffer(bytes int) error {
	if u, ok := c.PacketConn.(interface{ SetReadBuffer(int) error }); ok {
		return u.SetReadBuffer(bytes)
	}
	return nil
}

// SetWriteBuffer satisfies quic-go performance optimizations
func (c *ObfuscatedConn) SetWriteBuffer(bytes int) error {
	if u, ok := c.PacketConn.(interface{ SetWriteBuffer(int) error }); ok {
		return u.SetWriteBuffer(bytes)
	}
	return nil
}