package blockchain

import (
	"path/filepath"
	"testing"

	"github.com/sudharma-networks/sudharma/consensus"
)

type recordingProofVerifier struct {
	supported map[uint32]bool
	result    bool
	calls     int
}

func (v *recordingProofVerifier) SupportsVersion(version uint32) bool {
	return v != nil && v.supported[version]
}

func (v *recordingProofVerifier) Verify(*Block) bool {
	v.calls++
	return v.result
}

func TestNewChainWithConsensusRequiresPolicyCapabilities(t *testing.T) {
	policy := PoWPolicy{GPUV1ActivationHeight: 100}

	if _, err := NewChainWithConsensus(policy, nil); err == nil {
		t.Fatal("finite activation accepted a nil proof verifier")
	}
	legacy := &recordingProofVerifier{supported: map[uint32]bool{1: true}, result: true}
	if _, err := NewChainWithConsensus(policy, legacy); err == nil {
		t.Fatal("finite activation accepted a legacy-only proof verifier")
	}
	both := &recordingProofVerifier{supported: map[uint32]bool{1: true, 2: true}, result: true}
	chain, err := NewChainWithConsensus(policy, both)
	if err != nil {
		t.Fatal(err)
	}
	if got := chain.PoWPolicy(); got != policy {
		t.Fatalf("chain policy = %+v, want %+v", got, policy)
	}
}

func TestChainRejectsWrongVersionBeforeProofDispatch(t *testing.T) {
	verifier := &recordingProofVerifier{supported: map[uint32]bool{1: true, 2: true}, result: true}
	chain, err := NewChainWithConsensus(PoWPolicy{GPUV1ActivationHeight: 100}, verifier)
	if err != nil {
		t.Fatal(err)
	}

	block := nextPolicyTestBlock(chain.Tip(), 2)
	if err := chain.AddBlock(block); err == nil {
		t.Fatal("Version 2 block accepted before activation")
	}
	if verifier.calls != 0 {
		t.Fatalf("proof verifier called %d times for wrong version", verifier.calls)
	}
}

func TestChainUsesConfiguredProofVerifier(t *testing.T) {
	verifier := &recordingProofVerifier{supported: map[uint32]bool{1: true}, result: true}
	chain, err := NewChainWithConsensus(LegacyOnlyPoWPolicy(), verifier)
	if err != nil {
		t.Fatal(err)
	}

	block := nextPolicyTestBlock(chain.Tip(), 1)
	if err := chain.AddBlock(block); err != nil {
		t.Fatal(err)
	}
	if verifier.calls != 1 {
		t.Fatalf("proof verifier calls = %d, want 1", verifier.calls)
	}
}

func TestConsensusPolicySurvivesStorageCloneAndReplacement(t *testing.T) {
	policy := PoWPolicy{GPUV1ActivationHeight: 100}
	verifier := &recordingProofVerifier{supported: map[uint32]bool{1: true, 2: true}, result: true}
	chain, err := NewChainWithConsensus(policy, verifier)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "chain.json")
	if err := chain.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadChainFromFileWithConsensus(path, policy, verifier)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.PoWPolicy(); got != policy {
		t.Fatalf("loaded policy = %+v, want %+v", got, policy)
	}

	cloned, err := ValidateAndCloneChain(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if got := cloned.PoWPolicy(); got != policy {
		t.Fatalf("cloned policy = %+v, want %+v", got, policy)
	}

	candidate := NewChain()
	if err := loaded.ReplaceWith(candidate); err != nil {
		t.Fatal(err)
	}
	if got := loaded.PoWPolicy(); got != policy {
		t.Fatalf("replacement changed policy to %+v, want %+v", got, policy)
	}
}

func nextPolicyTestBlock(previous *Block, version uint32) *Block {
	block := &Block{
		Version:      version,
		Height:       previous.Height + 1,
		Timestamp:    previous.Timestamp + 1,
		PreviousHash: previous.Hash(),
		Difficulty: consensus.NextDifficultyFromHistory(
			previous.Difficulty,
			nil,
		),
		MinerAddress: "policy-test-miner",
	}
	block.UpdateMerkleRoot()
	return block
}
