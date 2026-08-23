package pow

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestHashBlock(t *testing.T) {
	block := blockchain.NewGenesisBlock()

	hash := HashBlock(block, 0)

	if len(hash) != 64 {
		t.Fatalf("expected 64-character hash, got %d", len(hash))
	}
}

func TestDifficultyTarget(t *testing.T) {
	target1 := TargetFromDifficulty(1)
	target100 := TargetFromDifficulty(100)

	if target100.Cmp(target1) >= 0 {
		t.Fatal("higher difficulty should produce a smaller target")
	}
}
