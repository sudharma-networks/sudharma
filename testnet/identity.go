package testnet

import (
	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/p2p"
)

// ProtocolNetworkID is the P2P namespace enforced by handshakes in this
// pre-mainnet release line. The final mainnet launch must intentionally change
// the P2P network ID and genesis identity so testnet and mainnet can never peer.
const ProtocolNetworkID = p2p.NetworkID

// GenesisHash returns the deterministic block-0 hash expected by this testnet
// software release. Operators can compare this fingerprint across seed nodes.
func GenesisHash() string {
	return blockchain.NewGenesisBlock().Hash()
}
