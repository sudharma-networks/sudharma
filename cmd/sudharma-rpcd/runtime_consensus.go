package main

import (
	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/pow"
)

// runtimeConsensusForNetwork composes the immutable network PoW policy with
// the Khushi-capable verification-only chain verifier. Network activation
// heights remain defined by params and are not changed here.
func runtimeConsensusForNetwork(network params.NetworkID) (blockchain.PoWPolicy, blockchain.ProofVerifier, error) {
	policy, err := blockchain.PoWPolicyForNetwork(network)
	if err != nil {
		return blockchain.PoWPolicy{}, nil, err
	}
	verifier, err := pow.NewChainProofVerifier(policy)
	if err != nil {
		return blockchain.PoWPolicy{}, nil, err
	}
	return policy, verifier, nil
}
