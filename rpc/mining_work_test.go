package rpc

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/pow"
)

func TestNewMiningWorkTemplateBindsImmutableFields(t *testing.T) {
	block := &blockchain.Block{
		Version:      2,
		Height:       7500,
		Timestamp:    1786924860,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   7,
		MinerAddress: "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
	}

	work, err := NewMiningWorkTemplate(block)
	if err != nil {
		t.Fatalf("create mining work: %v", err)
	}
	if work.WorkID == "" {
		t.Fatal("work ID must not be empty")
	}
	if work.Algorithm != pow.GPUV1AlgorithmID {
		t.Fatalf("algorithm mismatch: got %q", work.Algorithm)
	}
	if work.Version != 2 || work.Height != block.Height || work.Difficulty != block.Difficulty {
		t.Fatalf("unexpected consensus fields: %+v", work)
	}
	if work.RewardAddress != block.MinerAddress {
		t.Fatalf("reward address mismatch: got %q want %q", work.RewardAddress, block.MinerAddress)
	}
	if len(work.TargetHex) != 64 {
		t.Fatalf("target must be 32-byte hex: got %q", work.TargetHex)
	}
	if work.HeaderPrefixHex == "" {
		t.Fatal("header prefix must not be empty")
	}

	originalID := work.WorkID
	originalHeader := work.HeaderPrefixHex
	block.PreviousHash = "mutated"
	block.MerkleRoot = "mutated"
	block.MinerAddress = "mutated"
	block.Difficulty = 1
	if work.WorkID != originalID || work.HeaderPrefixHex != originalHeader {
		t.Fatal("work template changed after caller mutated source block")
	}
}

func TestNewMiningWorkTemplateWorkIDChangesWhenRewardAddressChanges(t *testing.T) {
	base := &blockchain.Block{
		Version:      2,
		Height:       7500,
		Timestamp:    1786924860,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   7,
		MinerAddress: "miner-a",
	}

	a, err := NewMiningWorkTemplate(base)
	if err != nil {
		t.Fatalf("create first work: %v", err)
	}
	changed := *base
	changed.MinerAddress = "miner-b"
	b, err := NewMiningWorkTemplate(&changed)
	if err != nil {
		t.Fatalf("create second work: %v", err)
	}
	if a.WorkID == b.WorkID {
		t.Fatal("work ID must bind reward address")
	}
}

func TestNewMiningWorkTemplateRejectsUnsupportedBlock(t *testing.T) {
	for _, block := range []*blockchain.Block{
		nil,
		{Version: 1, Height: 1, Difficulty: 1, MinerAddress: "miner"},
		{Version: 2, Height: 1, Difficulty: 0, MinerAddress: "miner"},
		{Version: 2, Height: 1, Difficulty: 1},
	} {
		if _, err := NewMiningWorkTemplate(block); err == nil {
			t.Fatalf("expected unsupported block to be rejected: %+v", block)
		}
	}
}
