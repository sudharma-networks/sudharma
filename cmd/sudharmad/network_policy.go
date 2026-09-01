package main

import (
	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

// loadChainForNetwork loads persisted chain data under the caller's explicit
// network identity. Runtime callers must obtain network through the launch-
// gated parser before reaching this helper.
func loadChainForNetwork(path string, network params.NetworkID) (*blockchain.Chain, error) {
	return blockchain.LoadChainFromFileFor(path, network)
}

// newGenesisStateForPolicy creates an empty state bound to the same monetary
// policy as the active chain/network.
func newGenesisStateForPolicy(policy params.MonetaryPolicy) *blockchain.State {
	return blockchain.NewStateFor(policy)
}
