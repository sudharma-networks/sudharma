package testnet

import (
	"fmt"
	"net"
	"strings"
)

const (
	Name              = "Sudharma Public Testnet 1"
	Slug              = "sudharma-testnet-1"
	DefaultP2PPort    = 28444
	DefaultRPCPort    = 28545
	DefaultDataDir    = "data-testnet-1"
	MinimumSeedNodes  = 2
)

// Profile is public, non-secret launch metadata. Seed addresses can be DNS
// names or host:port pairs once infrastructure is provisioned.
type Profile struct {
	Name       string   `json:"name"`
	Slug       string   `json:"slug"`
	P2PPort    int      `json:"p2p_port"`
	RPCPort    int      `json:"rpc_port"`
	DataDir    string   `json:"data_directory"`
	Seeds      []string `json:"seeds"`
}

func DefaultProfile() Profile {
	return Profile{Name: Name, Slug: Slug, P2PPort: DefaultP2PPort, RPCPort: DefaultRPCPort, DataDir: DefaultDataDir}
}

func (p Profile) Validate() error {
	if strings.TrimSpace(p.Name) == "" || strings.TrimSpace(p.Slug) == "" {
		return fmt.Errorf("testnet name and slug are required")
	}
	if p.P2PPort < 1024 || p.P2PPort > 65535 || p.RPCPort < 1024 || p.RPCPort > 65535 || p.P2PPort == p.RPCPort {
		return fmt.Errorf("testnet ports must be distinct user ports")
	}
	if strings.TrimSpace(p.DataDir) == "" {
		return fmt.Errorf("testnet data directory is required")
	}
	seen := map[string]struct{}{}
	for _, seed := range p.Seeds {
		seed = strings.TrimSpace(seed)
		if seed == "" { return fmt.Errorf("seed address cannot be empty") }
		if _, _, err := net.SplitHostPort(seed); err != nil { return fmt.Errorf("invalid seed %q: %w", seed, err) }
		if _, exists := seen[seed]; exists { return fmt.Errorf("duplicate seed %q", seed) }
		seen[seed] = struct{}{}
	}
	return nil
}
