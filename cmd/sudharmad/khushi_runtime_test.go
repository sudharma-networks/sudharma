package main

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

func TestSudharmadRuntimeConsensusUsesKhushiVerifierWithoutActivation(t *testing.T) {
	policy, verifier, err := runtimeConsensusForNetwork(params.NetworkPublicTestnet)
	if err != nil {
		t.Fatalf("runtime consensus: %v", err)
	}

	wantPolicy, err := blockchain.PoWPolicyForNetwork(params.NetworkPublicTestnet)
	if err != nil {
		t.Fatalf("network policy: %v", err)
	}
	if policy != wantPolicy {
		t.Fatalf("runtime policy = %+v, want %+v", policy, wantPolicy)
	}
	if policy.GPUV1ActivationHeight != params.GPUV1ActivationDisabled {
		t.Fatalf("runtime activation height = %d, want disabled", policy.GPUV1ActivationHeight)
	}
	if verifier == nil {
		t.Fatal("runtime proof verifier is nil")
	}
	if !verifier.SupportsVersion(1) || !verifier.SupportsVersion(2) {
		t.Fatal("runtime proof verifier must support both legacy Version 1 and Khushi Version 2")
	}
}

func TestSudharmadNewChainForNetworkKeepsMainnetLaunchGateClosed(t *testing.T) {
	if params.MainnetLaunchAuthorized {
		t.Fatal("test requires mainnet launch authorization to remain disabled")
	}
	if _, err := newChainForNetwork(params.NetworkMainnet); err == nil {
		t.Fatal("runtime mainnet chain creation bypassed the launch gate")
	}
}
