package main

import (
	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

// loadChainForNetwork loads persisted chain data under an explicit immutable
// network identity. Runtime callers must obtain network through the launch-gated
// params.ParseNetwork path before calling this helper.
func loadChainForNetwork(path string, network params.NetworkID) (*blockchain.Chain, error) {
	return blockchain.LoadChainFromFileFor(path, network)
}

// newGenesisStateForPolicy creates empty chain state bound to the active
// network's monetary policy instead of the public-testnet compatibility policy.
func newGenesisStateForPolicy(policy params.MonetaryPolicy) *blockchain.State {
	return blockchain.NewStateFor(policy)
}
