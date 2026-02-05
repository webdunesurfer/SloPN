package iputil

import (
	"net"
)

// GetDestinationIP extracts the destination IPv4 address from a raw packet
func GetDestinationIP(packet []byte) net.IP {
	if len(packet) < 20 {
		return nil
	}
	// IPv4 destination address is at bytes 16-19
	return net.IP(packet[16:20])
}

// GetSourceIP extracts the source IPv4 address from a raw packet
func GetSourceIP(packet []byte) net.IP {
	if len(packet) < 20 {
		return nil
	}
	// IPv4 source address is at bytes 12-15
	return net.IP(packet[12:15])
}
