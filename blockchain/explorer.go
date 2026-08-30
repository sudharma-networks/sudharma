package blockchain

import "github.com/sudharma-networks/sudharma/transactions"

// ConfirmedTransaction is an immutable explorer view of a transaction and the
// canonical block that currently contains it.
type ConfirmedTransaction struct {
	Transaction    *transactions.Transaction `json:"transaction"`
	BlockHeight    uint64                    `json:"block_height"`
	BlockHash      string                    `json:"block_hash"`
	BlockTimestamp int64                     `json:"block_timestamp"`
}

// BlockByHash returns one block from the current canonical chain by its exact
// lowercase hexadecimal hash.
func (c *Chain) BlockByHash(hash string) (*Block, bool) {
	if c == nil || hash == "" {
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, block := range c.blocks {
		if block != nil && block.Hash() == hash {
			return block, true
		}
	}
	return nil, false
}

// RecentBlocks returns canonical blocks newest first. If before is non-nil,
// only blocks with height strictly lower than *before are returned.
func (c *Chain) RecentBlocks(limit int, before *uint64) []*Block {
	if c == nil || limit <= 0 {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*Block, 0, limit)
	for i := len(c.blocks) - 1; i >= 0 && len(result) < limit; i-- {
		block := c.blocks[i]
		if block == nil {
			continue
		}
		if before != nil && block.Height >= *before {
			continue
		}
		result = append(result, block)
	}
	return result
}

// RecentTransactions returns confirmed transactions from the current
// canonical chain, scanning newest blocks first while preserving transaction
// order within each block. If beforeHeight is non-nil, only blocks below that
// height are scanned.
func (c *Chain) RecentTransactions(limit int, beforeHeight *uint64) []ConfirmedTransaction {
	if c == nil || limit <= 0 {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]ConfirmedTransaction, 0, limit)
	for i := len(c.blocks) - 1; i >= 0 && len(result) < limit; i-- {
		block := c.blocks[i]
		if block == nil {
			continue
		}
		if beforeHeight != nil && block.Height >= *beforeHeight {
			continue
		}
		blockHash := block.Hash()
		for _, tx := range block.Transactions {
			if tx == nil {
				continue
			}
			result = append(result, ConfirmedTransaction{
				Transaction:    tx,
				BlockHeight:    block.Height,
				BlockHash:      blockHash,
				BlockTimestamp: block.Timestamp,
			})
			if len(result) == limit {
				break
			}
		}
	}
	return result
}

// TransactionsForAddress returns confirmed transactions where address is the
// sender or recipient, newest blocks first. If beforeHeight is non-nil, only
// blocks below that height are scanned.
func (c *Chain) TransactionsForAddress(address string, limit int, beforeHeight *uint64) []ConfirmedTransaction {
	if c == nil || address == "" || limit <= 0 {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]ConfirmedTransaction, 0, limit)
	for i := len(c.blocks) - 1; i >= 0 && len(result) < limit; i-- {
		block := c.blocks[i]
		if block == nil {
			continue
		}
		if beforeHeight != nil && block.Height >= *beforeHeight {
			continue
		}
		blockHash := block.Hash()
		for _, tx := range block.Transactions {
			if tx == nil || (tx.From != address && tx.To != address) {
				continue
			}
			result = append(result, ConfirmedTransaction{
				Transaction:    tx,
				BlockHeight:    block.Height,
				BlockHash:      blockHash,
				BlockTimestamp: block.Timestamp,
			})
			if len(result) == limit {
				break
			}
		}
	}
	return result
}
