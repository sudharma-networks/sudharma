package blockchain

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
)

func validateBlockTransactions(block *Block) error {
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}
	if len(block.Transactions) > params.MaxBlockTransactions {
		return fmt.Errorf(
			"block contains too many transactions: %d > %d",
			len(block.Transactions),
			params.MaxBlockTransactions,
		)
	}

	totalBytes := 0
	for _, tx := range block.Transactions {
		if tx == nil {
			return fmt.Errorf("block contains nil transaction")
		}
		if err := transactions.ValidateResourceBounds(tx); err != nil {
			return fmt.Errorf("block transaction %s invalid: %w", tx.ID, err)
		}
		totalBytes += tx.EstimatedSerializedSize()
	}
	if totalBytes > params.MaxBlockTransactionBytes {
		return fmt.Errorf(
			"block transaction payload exceeds maximum size: %d > %d",
			totalBytes,
			params.MaxBlockTransactionBytes,
		)
	}
	return nil
}
