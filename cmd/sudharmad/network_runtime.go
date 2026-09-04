package main

import (
	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

// loadChainForNetwork loads persisted chain data under an explicit immutable
// network identity and Khushi-capable verifier. Runtime callers must obtain
// network through the launch-gated params.ParseNetwork path before calling
// this helper.
func loadChainForNetwork(path string, network params.NetworkID) (*blockchain.Chain, error) {
	policy, verifier, err := runtimeConsensusForNetwork(network)
	if err != nil {
		return nil, err
	}
	return blockchain.LoadChainFromFileForWithConsensus(path, network, policy, verifier)
}

// newChainForNetwork creates the active runtime chain with the same immutable
// network policy and Khushi-capable verifier used for persisted-chain loads.
// NewChainForWithConsensus still enforces the existing mainnet launch gate.
func newChainForNetwork(network params.NetworkID) (*blockchain.Chain, error) {
	policy, verifier, err := runtimeConsensusForNetwork(network)
	if err != nil {
		return nil, err
	}
	return blockchain.NewChainForWithConsensus(network, policy, verifier)
}

// newGenesisStateForPolicy creates empty chain state bound to the active
// network's monetary policy instead of the public-testnet compatibility policy.
func newGenesisStateForPolicy(policy params.MonetaryPolicy) *blockchain.State {
	return blockchain.NewStateFor(policy)
}
