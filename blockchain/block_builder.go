package blockchain

import (
	"fmt"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
)

// NewBlockFromMempool creates a candidate block from pending transactions while
// honoring the same count and transaction-payload byte limits enforced by
// consensus validation.
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

	pending := pool.AllTransactions()
	selected := make([]*transactions.Transaction, 0, min(len(pending), params.MaxBlockTransactions))
	totalBytes := 0
	for _, tx := range pending {
		if tx == nil {
			continue
		}
		if len(selected) >= params.MaxBlockTransactions {
			break
		}
		size := tx.EstimatedSerializedSize()
		if totalBytes+size > params.MaxBlockTransactionBytes {
			continue
		}
		selected = append(selected, tx)
		totalBytes += size
	}

	block := &Block{
		Version:      1,
		Height:       previousBlock.Height + 1,
		Timestamp:    time.Now().Unix(),
		PreviousHash: previousBlock.Hash(),
		Difficulty:   previousBlock.Difficulty,
		Nonce:        0,
		Transactions: selected,
	}

	block.UpdateMerkleRoot()
	return block, nil
}
