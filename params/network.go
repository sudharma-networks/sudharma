package params

import "fmt"

// NetworkID is the P2P / genesis namespace. Public testnet and mainnet must
// never share an identity, or they could peer and mix monetary policy.
type NetworkID string

const (
	NetworkPublicTestnet NetworkID = "sudharma-testnet-1"
	NetworkMainnet       NetworkID = "sudharma-mainnet-1"

	// DefaultNetwork is the live public-testnet identity. Binaries must keep
	// this default until an explicit launch decision arms mainnet.
	DefaultNetwork = NetworkPublicTestnet

	// MainnetLaunchAuthorized is the hard gate. It must remain false until
	// genesis timestamp freeze, security audit, seed topology, and a human
	// launch decision are complete.
	MainnetLaunchAuthorized = false

	// MainnetGenesisTimestamp is unset (0) until the launch freeze. The
	// candidate genesis still uses this value so reviewers can lock a hash
	// without authorizing nodes to join mainnet.
	MainnetGenesisTimestamp uint64 = 0
)

// ParseNetwork maps operator-facing names onto a network identity.
// "mainnet" is rejected while launch is unauthorized.
func ParseNetwork(raw string) (NetworkID, error) {
	switch raw {
	case "", "public-testnet", "testnet", string(NetworkPublicTestnet):
		return NetworkPublicTestnet, nil
	case "mainnet", string(NetworkMainnet):
		if !MainnetLaunchAuthorized {
			return "", fmt.Errorf("mainnet launch is not authorized")
		}
		return NetworkMainnet, nil
	default:
		return "", fmt.Errorf("unknown network %q", raw)
	}
}

// MonetaryPolicyFor returns the monetary policy that belongs to a network.
func MonetaryPolicyFor(network NetworkID) (MonetaryPolicy, error) {
	switch network {
	case NetworkPublicTestnet:
		return MonetaryPolicyPublicTestnet, nil
	case NetworkMainnet:
		return MonetaryPolicyMainnet, nil
	default:
		return 0, fmt.Errorf("unknown network %q", network)
	}
}
