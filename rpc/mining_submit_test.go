package rpc

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func testMiningBlock(height uint64, reward string) *blockchain.Block {
	return &blockchain.Block{
		Version:      2,
		Height:       height,
		Timestamp:    1786924860 + height,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   1,
		MinerAddress: reward,
	}
}

func solutionFor(work MiningWorkTemplate, nonce uint64) MiningSolution {
	return MiningSolution{
		WorkID:          work.WorkID,
		Nonce:           nonce,
		Algorithm:       work.Algorithm,
		Version:         work.Version,
		Height:          work.Height,
		HeaderPrefixHex: work.HeaderPrefixHex,
		RewardAddress:   work.RewardAddress,
	}
}

func TestMiningWorkServiceAcceptsValidSolution(t *testing.T) {
	service := NewMiningWorkService(func(block *blockchain.Block, nonce uint64) bool {
		return block.Height == 7500 && nonce == 42
	})
	work, err := service.Issue(testMiningBlock(7500, "miner-a"))
	if err != nil {
		t.Fatalf("issue work: %v", err)
	}

	result := service.Submit(solutionFor(work, 42))
	if result.Status != MiningSubmitAccepted {
		t.Fatalf("expected accepted solution, got %+v", result)
	}
}

func TestMiningWorkServiceRejectsInvalidSolution(t *testing.T) {
	service := NewMiningWorkService(func(_ *blockchain.Block, _ uint64) bool { return false })
	work, err := service.Issue(testMiningBlock(7500, "miner-a"))
	if err != nil {
		t.Fatalf("issue work: %v", err)
	}

	result := service.Submit(solutionFor(work, 7))
	if result.Status != MiningSubmitInvalid {
		t.Fatalf("expected invalid solution rejection, got %+v", result)
	}
}

func TestMiningWorkServiceRejectsStaleSolution(t *testing.T) {
	service := NewMiningWorkService(func(_ *blockchain.Block, _ uint64) bool { return true })
	oldWork, err := service.Issue(testMiningBlock(7500, "miner-a"))
	if err != nil {
		t.Fatalf("issue old work: %v", err)
	}
	if _, err := service.Issue(testMiningBlock(7501, "miner-a")); err != nil {
		t.Fatalf("issue replacement work: %v", err)
	}

	result := service.Submit(solutionFor(oldWork, 42))
	if result.Status != MiningSubmitStale {
		t.Fatalf("expected stale solution rejection, got %+v", result)
	}
}

func TestMiningWorkServiceRejectsMutatedImmutableFields(t *testing.T) {
	service := NewMiningWorkService(func(_ *blockchain.Block, _ uint64) bool { return true })
	work, err := service.Issue(testMiningBlock(7500, "miner-a"))
	if err != nil {
		t.Fatalf("issue work: %v", err)
	}

	mutations := []func(*MiningSolution){
		func(s *MiningSolution) { s.Algorithm = "other" },
		func(s *MiningSolution) { s.Version++ },
		func(s *MiningSolution) { s.Height++ },
		func(s *MiningSolution) { s.HeaderPrefixHex = "00" + s.HeaderPrefixHex },
		func(s *MiningSolution) { s.RewardAddress = "miner-b" },
	}
	for i, mutate := range mutations {
		solution := solutionFor(work, 42)
		mutate(&solution)
		result := service.Submit(solution)
		if result.Status != MiningSubmitMutated {
			t.Fatalf("mutation %d: expected mutated-work rejection, got %+v", i, result)
		}
	}
}
