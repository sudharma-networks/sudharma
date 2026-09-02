package blockchain

import (
	"path/filepath"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestLoadChainFromFileForWithConsensusPreservesExplicitPolicy(t *testing.T) {
	policy := PoWPolicy{GPUV1ActivationHeight: 100}
	verifier := &recordingProofVerifier{
		supported: map[uint32]bool{1: true, 2: true},
		result:    true,
	}
	chain, err := NewChainForWithConsensus(
		params.NetworkPublicTestnet,
		policy,
		verifier,
	)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "chain.json")
	if err := chain.SaveToFile(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadChainFromFileForWithConsensus(
		path,
		params.NetworkPublicTestnet,
		policy,
		verifier,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.PoWPolicy(); got != policy {
		t.Fatalf("PoW policy = %+v, want %+v", got, policy)
	}
}

func TestLoadChainFromFileForWithConsensusRejectsIncapableVerifier(t *testing.T) {
	chain, err := NewChainFor(params.NetworkPublicTestnet)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "chain.json")
	if err := chain.SaveToFile(path); err != nil {
		t.Fatal(err)
	}

	policy := PoWPolicy{GPUV1ActivationHeight: 100}
	legacyOnly := &recordingProofVerifier{
		supported: map[uint32]bool{1: true},
		result:    true,
	}
	if _, err := LoadChainFromFileForWithConsensus(
		path,
		params.NetworkPublicTestnet,
		policy,
		legacyOnly,
	); err == nil {
		t.Fatal("finite policy accepted verifier without Version 2 support")
	}
}

func TestLoadChainFromFileForUsesDisabledNetworkPolicy(t *testing.T) {
	chain, err := NewChainFor(params.NetworkPublicTestnet)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "chain.json")
	if err := chain.SaveToFile(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadChainFromFileFor(path, params.NetworkPublicTestnet)
	if err != nil {
		t.Fatal(err)
	}
	want, err := PoWPolicyForNetwork(params.NetworkPublicTestnet)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.PoWPolicy(); got != want {
		t.Fatalf("PoW policy = %+v, want %+v", got, want)
	}
	if got.GPUV1ActivationHeight != params.GPUV1ActivationDisabled {
		t.Fatalf("activation height = %d, want disabled", got.GPUV1ActivationHeight)
	}
}
