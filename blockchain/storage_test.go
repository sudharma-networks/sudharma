package blockchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sudharma-networks/sudharma/consensus"
)

func TestSaveAndLoadChain(t *testing.T) {
	chain := NewChain()

	previous := chain.Tip()

	blockTime := int64(60)

	difficulty := consensus.NextDifficulty(
		previous.Difficulty,
		blockTime,
	)

	block := &Block{
		Version:      1,
		Height:       previous.Height + 1,
		Timestamp:    previous.Timestamp + blockTime,
		PreviousHash: previous.Hash(),
		Difficulty:   difficulty,
		Nonce:        0,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	if !mineTestBlock(
		block,
		1_000_000,
	) {
		t.Fatal(
			"failed to mine test block",
		)
	}

	if err := chain.AddBlock(
		block,
	); err != nil {
		t.Fatal(err)
	}

	tempDir := t.TempDir()

	path := filepath.Join(
		tempDir,
		"sudharma-chain.json",
	)

	if err := chain.SaveToFile(
		path,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf(
			"blockchain file was not created: %v",
			err,
		)
	}

	loaded, err := LoadChainFromFile(path)

	if err != nil {
		t.Fatal(err)
	}

	if loaded.Height() != chain.Height() {
		t.Fatalf(
			"expected loaded height %d, got %d",
			chain.Height(),
			loaded.Height(),
		)
	}

	if loaded.Tip().Hash() != chain.Tip().Hash() {
		t.Fatal(
			"loaded chain tip does not match saved chain",
		)
	}

	if loaded.TotalWork().Cmp(
		chain.TotalWork(),
	) != 0 {
		t.Fatal(
			"loaded chain work does not match saved chain",
		)
	}
}

func TestCorruptedChainFileRejected(t *testing.T) {
	tempDir := t.TempDir()

	path := filepath.Join(
		tempDir,
		"corrupted.json",
	)

	if err := os.WriteFile(
		path,
		[]byte("not-valid-json"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	_, err := LoadChainFromFile(path)

	if err == nil {
		t.Fatal(
			"corrupted blockchain file was accepted",
		)
	}
}
