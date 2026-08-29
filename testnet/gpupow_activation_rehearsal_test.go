package testnet

import (
	"path/filepath"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

type rehearsalProofVerifier struct{}

func (rehearsalProofVerifier) SupportsVersion(version uint32) bool {
	return version == 1 || version == 2
}

func (rehearsalProofVerifier) Verify(*blockchain.Block) bool { return true }

func TestGPUActivationRollbackRehearsal(t *testing.T) {
	policy := blockchain.PoWPolicy{GPUV1ActivationHeight: 3}
	first := newRehearsalChain(t, policy)
	second := newRehearsalChain(t, policy)
	legacy := blockchain.NewChain()

	early := buildRehearsalBlock(t, first, 2, "early-gpu")
	if err := first.AddBlock(early); err == nil {
		t.Fatal("upgraded node accepted Version 2 before activation")
	}

	for height := uint64(1); height < policy.GPUV1ActivationHeight; height++ {
		block := buildRehearsalBlock(t, first, 1, "legacy-miner")
		for name, chain := range map[string]*blockchain.Chain{
			"first upgraded":  first,
			"second upgraded": second,
			"legacy observer": legacy,
		} {
			if err := chain.AddBlock(block); err != nil {
				t.Fatalf("%s rejected Version 1 height %d: %v", name, height, err)
			}
		}
	}

	boundary := buildRehearsalBlock(t, first, 2, "gpu-miner")
	if err := first.AddBlock(boundary); err != nil {
		t.Fatalf("first upgraded node rejected boundary block: %v", err)
	}
	if err := second.AddBlock(boundary); err != nil {
		t.Fatalf("second upgraded node rejected boundary block: %v", err)
	}
	if err := legacy.AddBlock(boundary); err == nil {
		t.Fatal("legacy observer followed Version 2 across the boundary")
	}
	if legacy.Height() != policy.GPUV1ActivationHeight-1 {
		t.Fatalf("legacy observer height = %d, want %d", legacy.Height(), policy.GPUV1ActivationHeight-1)
	}

	path := filepath.Join(t.TempDir(), "rehearsal-chain.json")
	if err := second.SaveToFile(path); err != nil {
		t.Fatalf("save upgraded chain: %v", err)
	}
	restarted, err := blockchain.LoadChainFromFileWithConsensus(path, policy, rehearsalProofVerifier{})
	if err != nil {
		t.Fatalf("restart upgraded chain: %v", err)
	}
	if restarted.Height() != second.Height() || restarted.Tip().Hash() != second.Tip().Hash() {
		t.Fatal("upgraded restart did not replay the active Version 2 chain")
	}

	candidate := newRehearsalChain(t, policy)
	for height := uint64(1); height < policy.GPUV1ActivationHeight; height++ {
		block, ok := first.BlockByHeight(height)
		if !ok {
			t.Fatalf("shared block %d missing", height)
		}
		if err := candidate.AddBlock(block); err != nil {
			t.Fatalf("candidate rejected shared block %d: %v", height, err)
		}
	}
	if err := candidate.AddBlock(buildRehearsalBlock(t, candidate, 2, "fork-gpu-miner")); err != nil {
		t.Fatalf("candidate boundary block: %v", err)
	}
	if err := candidate.AddBlock(buildRehearsalBlock(t, candidate, 2, "fork-gpu-miner")); err != nil {
		t.Fatalf("candidate post-boundary block: %v", err)
	}

	state := blockchain.NewState()
	for height := uint64(1); height <= first.Height(); height++ {
		block, ok := first.BlockByHeight(height)
		if !ok {
			t.Fatalf("current block %d missing", height)
		}
		if _, err := blockchain.ProcessBlock(state, block, block.MinerAddress); err != nil {
			t.Fatalf("process current block %d: %v", height, err)
		}
	}
	adopted, err := blockchain.ReorganizeToCandidate(first, state, candidate)
	if err != nil {
		t.Fatalf("shallow Version 2 reorganization: %v", err)
	}
	if !adopted || first.Tip().Hash() != candidate.Tip().Hash() {
		t.Fatal("better shallow Version 2 fork was not adopted")
	}
}

func newRehearsalChain(t *testing.T, policy blockchain.PoWPolicy) *blockchain.Chain {
	t.Helper()
	chain, err := blockchain.NewChainWithConsensus(policy, rehearsalProofVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	return chain
}

func buildRehearsalBlock(
	t *testing.T,
	chain *blockchain.Chain,
	version uint32,
	minerAddress string,
) *blockchain.Block {
	t.Helper()
	previous := chain.Tip()
	difficulty, err := blockchain.ExpectedNextDifficulty(chain)
	if err != nil {
		t.Fatal(err)
	}
	block := &blockchain.Block{
		Version:      version,
		Height:       previous.Height + 1,
		Timestamp:    previous.Timestamp + 1,
		PreviousHash: previous.Hash(),
		Difficulty:   difficulty,
		MinerAddress: minerAddress,
	}
	block.UpdateMerkleRoot()
	return block
}
