package pool

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/pow"
)

func testBlock(t *testing.T) *blockchain.Block {
	t.Helper()
	genesis := blockchain.NewGenesisBlock()
	block, err := blockchain.NewBlockFromMempool(genesis, mempool.NewMempool())
	if err != nil {
		t.Fatal(err)
	}
	block.Difficulty = 1
	block.MinerAddress = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	block.UpdateMerkleRoot()
	return block
}

func findNoncePoolOnly(t *testing.T, block *blockchain.Block, poolDifficulty, blockDifficulty uint32) uint64 {
	t.Helper()
	for nonce := uint64(0); nonce < 5_000_000; nonce++ {
		hash := pow.HashBlock(block, nonce)
		if pow.ValidHash(hash, poolDifficulty) && !pow.ValidHash(hash, blockDifficulty) {
			return nonce
		}
	}
	t.Fatalf("no pool-only nonce found for pool=%d block=%d", poolDifficulty, blockDifficulty)
	return 0
}

func findNonce(t *testing.T, block *blockchain.Block, difficulty uint32) uint64 {
	t.Helper()
	for nonce := uint64(0); nonce < 5_000_000; nonce++ {
		hash := pow.HashBlock(block, nonce)
		if pow.ValidHash(hash, difficulty) {
			return nonce
		}
	}
	t.Fatalf("no nonce found for difficulty %d", difficulty)
	return 0
}

func TestValidateShareAcceptsPoolShareAndBlock(t *testing.T) {
	block := testBlock(t)
	block.Difficulty = 100
	blockShare := findNoncePoolOnly(t, block, 1, 100)

	result, err := ValidateShare(block, blockShare, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != ShareValid {
		t.Fatalf("kind = %s", result.Kind)
	}

	blockNonce := findNonce(t, block, 100)
	blockResult, err := ValidateShare(block, blockNonce, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	if blockResult.Kind != ShareBlock {
		t.Fatalf("kind = %s", blockResult.Kind)
	}
}

func TestValidateShareRejectsLowDifficultyShare(t *testing.T) {
	block := testBlock(t)
	block.Difficulty = 10_000
	result, err := ValidateShare(block, 1, 5_000, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != ShareInvalid {
		t.Fatalf("kind = %s", result.Kind)
	}
}

func TestShareValueUsesDifficultyRatio(t *testing.T) {
	reward := consensus.BlockSubsidy(1)
	got := ShareValue(reward, 1, 100)
	want := reward / 100
	if got != want {
		t.Fatalf("share value = %d want %d", got, want)
	}
}
