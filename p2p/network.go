package p2p

import "github.com/sudharma-networks/sudharma/params"

// localNetworkID is the P2P namespace this process advertises in handshakes.
// Default remains public-testnet until SetLocalNetworkID is called at startup.
var localNetworkID = NetworkID

// SetLocalNetworkID selects the handshake namespace for the active network.
func SetLocalNetworkID(network params.NetworkID) {
	switch network {
	case params.NetworkMainnet:
		localNetworkID = MainnetNetworkID
	case params.NetworkPublicTestnet:
		localNetworkID = NetworkID
	default:
		localNetworkID = string(network)
	}
}

// LocalNetworkID returns the handshake namespace in use by this process.
func LocalNetworkID() string {
	return localNetworkID
}

// ResetLocalNetworkIDForTests restores the default public-testnet namespace.
func ResetLocalNetworkIDForTests() {
	localNetworkID = NetworkID
}
