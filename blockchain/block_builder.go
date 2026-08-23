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

	if previousBlock == nil {
		return nil, fmt.Errorf("previous block cannot be nil")
	}

	if pool == nil {
		return nil, fmt.Errorf("mempool cannot be nil")
	}

	block := &Block{
		Version:      1,
		Height:       previousBlock.Height + 1,
		Timestamp:    time.Now().Unix(),
		PreviousHash: previousBlock.Hash(),
		Difficulty:   previousBlock.Difficulty,
		Nonce:        0,
		Transactions: pool.AllTransactions(),
	}

	block.UpdateMerkleRoot()

	return block, nil
}
