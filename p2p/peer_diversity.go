package p2p

import (
	"fmt"
	"net"
	"strings"
)

const MaxPeersPerNetworkGroup = 2

// PeerNetworkGroup returns a coarse network group used to limit
// concentration of peers from the same address range.
//
// IPv4 peers are grouped by /16. IPv6 peers are grouped by the first
// 32 bits. Hostnames are grouped by normalized hostname.
func PeerNetworkGroup(address string) (string, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", fmt.Errorf("peer address cannot be empty")
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf("invalid peer address %q: %w", address, err)
	}

	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if host == "" {
		return "", fmt.Errorf("peer host cannot be empty")
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return "dns:" + strings.ToLower(host), nil
	}

	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("ipv4:%d.%d", v4[0], v4[1]), nil
	}

	v6 := ip.To16()
	if v6 == nil {
		return "", fmt.Errorf("invalid peer IP %q", host)
	}

	return fmt.Sprintf("ipv6:%02x%02x:%02x%02x", v6[0], v6[1], v6[2], v6[3]), nil
}

// CanAddPeerFromNetworkGroup checks whether adding address would exceed
// the per-network-group diversity limit among existing peer addresses.
func CanAddPeerFromNetworkGroup(existing []string, address string) bool {
	group, err := PeerNetworkGroup(address)
	if err != nil {
		return false
	}

	count := 0
	for _, existingAddress := range existing {
		existingGroup, err := PeerNetworkGroup(existingAddress)
		if err != nil {
			continue
		}
		if existingGroup == group {
			count++
		}
	}

	return count < MaxPeersPerNetworkGroup
}
