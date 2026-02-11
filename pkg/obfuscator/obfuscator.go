package obfuscator

import (
	"crypto/sha256"
	"net"
)

// ObfuscatedConn wraps a net.PacketConn and XORs all packets
type ObfuscatedConn struct {
	net.PacketConn
	key []byte
}

// NewObfuscatedConn creates a new XOR-masked connection
func NewObfuscatedConn(conn net.PacketConn, secret string) *ObfuscatedConn {
	h := sha256.Sum256([]byte(secret))
	return &ObfuscatedConn{
		PacketConn: conn,
		key:        h[:],
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

// WriteTo masks outgoing packets
func (c *ObfuscatedConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	// We must copy because quic-go might reuse the buffer
	buf := make([]byte, len(p))
	copy(buf, p)
	c.xor(buf)
	return c.PacketConn.WriteTo(buf, addr)
}
