package iputil

import (
	"encoding/hex"
	"fmt"
	"net"
)

// Protocol constants
const (
	ProtoICMP = 1
	ProtoTCP  = 6
	ProtoUDP  = 17
)

// detectOffset finds where the IPv4 header starts (0x45)
func detectOffset(packet []byte) int {
	for i := 0; i < len(packet)-20; i++ {
		if packet[i] == 0x45 {
			return i
		}
	}
	return 0
}

// GetDestinationIP extracts the destination IPv4 address from a raw packet
func GetDestinationIP(packet []byte) net.IP {
	offset := detectOffset(packet)
	if len(packet) < 20+offset {
		return nil
	}
	return net.IP(packet[16+offset : 20+offset])
}

// GetSourceIP extracts the source IPv4 address from a raw packet
func GetSourceIP(packet []byte) net.IP {
	offset := detectOffset(packet)
	if len(packet) < 20+offset {
		return nil
	}
	return net.IP(packet[12+offset : 16+offset])
}

// GetProtocol returns the protocol number from the IPv4 header
func GetProtocol(packet []byte) int {
	offset := detectOffset(packet)
	if len(packet) < 20+offset {
		return -1
	}
	return int(packet[9+offset])
}

// FormatPacketSummary returns a one-line summary of the packet
func FormatPacketSummary(packet []byte) string {
	offset := detectOffset(packet)
	src := GetSourceIP(packet)
	dst := GetDestinationIP(packet)
	proto := GetProtocol(packet)

	if src == nil || dst == nil {
		return fmt.Sprintf("len=%d", len(packet))
	}

	summary := fmt.Sprintf("%s -> %s ", src, dst)
	switch proto {
	case ProtoICMP:
		summary += "ICMP"
	case ProtoTCP:
		summary += "TCP"
	case ProtoUDP:
		summary += "UDP"
	default:
		summary += fmt.Sprintf("P(%d)", proto)
	}
	if offset > 0 {
		summary += fmt.Sprintf(" [off:%d]", offset)
	}
	return summary
}

// StripHeader returns ONLY the raw IP packet
func StripHeader(packet []byte) []byte {
	offset := detectOffset(packet)
	return packet[offset:]
}

// AddHeader adds the correct 4-byte PI header for Linux
func AddHeader(packet []byte, isLinux bool) []byte {
	if isLinux {
		header := []byte{0x00, 0x00, 0x08, 0x00}
		res := make([]byte, len(header)+len(packet))
		copy(res, header)
		copy(res[len(header):], packet)
		return res
	}
	return packet
}

func HexDump(packet []byte) string {
	limit := len(packet)
	if limit > 32 {
		limit = 32
	}
	return hex.EncodeToString(packet[:limit])
}