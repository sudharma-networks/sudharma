package blockchain

import (
	"fmt"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain/mempool"
)

// NewBlockFromMempool creates a candidate block from pending transactions.
func NewBlockFromMempool(
	previousBlock *Block,
	pool *mempool.Mempool,
) (*Block, error) {
	return NewBlockFromMempoolWithPolicy(
		previousBlock,
		pool,
		LegacyOnlyPoWPolicy(),
	)
}

// NewBlockFromMempoolWithPolicy creates a candidate using the block version
// selected by the chain's immutable proof-of-work policy.
func NewBlockFromMempoolWithPolicy(
	previousBlock *Block,
	pool *mempool.Mempool,
	policy PoWPolicy,
) (*Block, error) {

	if previousBlock == nil {
		return nil, fmt.Errorf("previous block cannot be nil")
	}

	if pool == nil {
		return nil, fmt.Errorf("mempool cannot be nil")
	}

	height := previousBlock.Height + 1
	version, err := policy.VersionAtHeight(height)
	if err != nil {
		return nil, fmt.Errorf("select block version: %w", err)
	}

	block := &Block{
		Version:      version,
		Height:       height,
		Timestamp:    time.Now().Unix(),
		PreviousHash: previousBlock.Hash(),
		Difficulty:   previousBlock.Difficulty,
		Nonce:        0,
		Transactions: pool.AllTransactions(),
	}

	block.UpdateMerkleRoot()

	return block, nil
}
