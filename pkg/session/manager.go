package session

import (
	"fmt"
	"net"
	"sync"

	"github.com/quic-go/quic-go"
)

// Session represents an active client connection
type Session struct {
	Conn *quic.Conn
	VIP  net.IP
}

// Manager handles all active client sessions and IP allocation
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session // Key: VIP string (e.g., "10.100.0.2")
	ipPool   chan net.IP
	serverIP net.IP
}

// NewManager creates a new session manager with a pool of available IPs
func NewManager(subnet string, serverIP string) (*Manager, error) {
	sIP := net.ParseIP(serverIP)
	if sIP == nil {
		return nil, fmt.Errorf("invalid server IP: %s", serverIP)
	}

	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, err
	}

	pool := make(chan net.IP, 253) // Max for /24
	
	// Populate pool, skipping network, broadcast, and server IP
	for ip := ipNet.IP.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
		currentIP := make(net.IP, len(ip))
		copy(currentIP, ip)
		
		// Skip .0 (network) and .255 (broadcast) - simple check for /24
		if currentIP[3] == 0 || currentIP[3] == 255 || currentIP.Equal(sIP) {
			continue
		}
		pool <- currentIP
	}

	return &Manager{
		sessions: make(map[string]*Session),
		ipPool:   pool,
		serverIP: sIP,
	}, nil
}

// AllocateIP gets an available IP from the pool
func (m *Manager) AllocateIP() (net.IP, error) {
	select {
	case ip := <-m.ipPool:
		return ip, nil
	default:
		return nil, fmt.Errorf("IP pool exhausted")
	}
}

// ReleaseIP returns an IP to the pool
func (m *Manager) ReleaseIP(ip net.IP) {
	m.ipPool <- ip
}

// AddSession registers a new client
func (m *Manager) AddSession(vip net.IP, conn *quic.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[vip.String()] = &Session{
		Conn: conn,
		VIP:  vip,
	}
}

// RemoveSession unregisters a client and releases its IP
func (m *Manager) RemoveSession(vip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[vip]; ok {
		m.ReleaseIP(net.ParseIP(vip))
		delete(m.sessions, vip)
	}
}

// GetSession returns the connection for a given VIP
func (m *Manager) GetSession(vip string) (*quic.Conn, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[vip]
	if !ok {
		return nil, false
	}
	return s.Conn, true
}

// GetServerIP returns the server's virtual IP
func (m *Manager) GetServerIP() net.IP {
	return m.serverIP
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
