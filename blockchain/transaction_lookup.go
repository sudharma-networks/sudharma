package blockchain

import "github.com/sudharma-networks/sudharma/transactions"

// TransactionByID returns a confirmed transaction and the block that contains
// it. Blocks and their transactions are immutable after admission to the chain,
// so returning these references is safe for read-only RPC inspection.
func (c *Chain) TransactionByID(txID string) (*transactions.Transaction, *Block, bool) {
	if c == nil || txID == "" {
		return nil, nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, block := range c.blocks {
		if block == nil {
			continue
		}
		for _, tx := range block.Transactions {
			if tx != nil && tx.ID == txID {
				return tx, block, true
			}
		}
	}
	return nil, nil, false
}
