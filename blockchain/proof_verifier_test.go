package blockchain

import (
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

type recordingProofVerifier struct {
	supported map[uint32]bool
	result    bool
	calls     int
}

func (v *recordingProofVerifier) SupportsVersion(version uint32) bool {
	return v != nil && v.supported[version]
}

func (v *recordingProofVerifier) Verify(block *Block) bool {
	if v == nil {
		return false
	}
	v.calls++
	return v.result
}

func TestNewChainForWithConsensusRequiresPolicyCapabilities(t *testing.T) {
	policy := PoWPolicy{GPUV1ActivationHeight: 100}

	if _, err := NewChainForWithConsensus(params.NetworkPublicTestnet, policy, nil); err == nil {
		t.Fatal("finite activation accepted nil verifier")
	}

	legacyOnly := &recordingProofVerifier{
		supported: map[uint32]bool{1: true},
		result:    true,
	}
	if _, err := NewChainForWithConsensus(params.NetworkPublicTestnet, policy, legacyOnly); err == nil {
		t.Fatal("finite activation accepted verifier without Version 2 support")
	}

	versioned := &recordingProofVerifier{
		supported: map[uint32]bool{1: true, 2: true},
		result:    true,
	}
	chain, err := NewChainForWithConsensus(params.NetworkPublicTestnet, policy, versioned)
	if err != nil {
		t.Fatal(err)
	}
	if chain.Network() != params.NetworkPublicTestnet {
		t.Fatalf("network = %q", chain.Network())
	}
	if got := chain.PoWPolicy(); got != policy {
		t.Fatalf("PoW policy = %+v, want %+v", got, policy)
	}
}

func TestNewChainForUsesDisabledNetworkPoWPolicy(t *testing.T) {
	chain, err := NewChainFor(params.NetworkPublicTestnet)
	if err != nil {
		t.Fatal(err)
	}
	want, err := PoWPolicyForNetwork(params.NetworkPublicTestnet)
	if err != nil {
		t.Fatal(err)
	}
	if got := chain.PoWPolicy(); got != want {
		t.Fatalf("PoW policy = %+v, want %+v", got, want)
	}
}

func TestChainRejectsWrongVersionBeforeProofDispatch(t *testing.T) {
	verifier := &recordingProofVerifier{
		supported: map[uint32]bool{1: true, 2: true},
		result:    true,
	}
	chain, err := NewChainForWithConsensus(
		params.NetworkPublicTestnet,
		PoWPolicy{GPUV1ActivationHeight: 100},
		verifier,
	)
	if err != nil {
		t.Fatal(err)
	}

	block := nextProofPolicyTestBlock(t, chain, 2)
	if err := chain.AddBlock(block); err == nil {
		t.Fatal("Version 2 block before activation was accepted")
	} else if !strings.Contains(err.Error(), "block version") {
		t.Fatalf("unexpected error: %v", err)
	}
	if verifier.calls != 0 {
		t.Fatalf("verifier calls = %d, want 0 for policy-rejected block", verifier.calls)
	}
}

func TestChainUsesConfiguredProofVerifier(t *testing.T) {
	verifier := &recordingProofVerifier{
		supported: map[uint32]bool{1: true},
		result:    true,
	}
	chain, err := NewChainForWithConsensus(
		params.NetworkPublicTestnet,
		LegacyOnlyPoWPolicy(),
		verifier,
	)
	if err != nil {
		t.Fatal(err)
	}

	block := nextProofPolicyTestBlock(t, chain, 1)
	if err := chain.AddBlock(block); err != nil {
		t.Fatalf("AddBlock with accepting verifier failed: %v", err)
	}
	if verifier.calls != 1 {
		t.Fatalf("verifier calls = %d, want 1", verifier.calls)
	}
}

func nextProofPolicyTestBlock(t *testing.T, chain *Chain, version uint32) *Block {
	t.Helper()
	previous := chain.Tip()
	if previous == nil {
		t.Fatal("chain tip is nil")
	}
	difficulty, err := ExpectedNextDifficulty(chain)
	if err != nil {
		t.Fatal(err)
	}
	block := &Block{
		Version:      version,
		Height:       previous.Height + 1,
		Timestamp:    previous.Timestamp + 1,
		PreviousHash: previous.Hash(),
		Difficulty:   difficulty,
		Nonce:        0,
		MinerAddress: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	block.UpdateMerkleRoot()
	return block
}
