package blockchain

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/transactions"
)

// ProcessBlock processes an entire Sudharma Network block atomically.
//
// Transactions are first applied to a temporary copy of the state.
// If ANY transaction fails, the real blockchain state remains unchanged.
//
// If every transaction succeeds, the miner receives:
//
//	block subsidy + miner transaction fees
//
// The temporary state is then committed to the real state.
func ProcessBlock(
	state *State,
	block *Block,
	minerAddress string,
) (uint64, error) {

	if state == nil {
		return 0, fmt.Errorf("state cannot be nil")
	}

	if block == nil {
		return 0, fmt.Errorf("block cannot be nil")
	}

	if minerAddress == "" {
		return 0, fmt.Errorf("miner address cannot be empty")
	}

	// Create temporary state.
	// Nothing touches the real state until the entire block succeeds.
	workingState := state.Clone()

	var totalMinerFees uint64

	for _, tx := range block.Transactions {
		if tx == nil {
			return 0, fmt.Errorf("block contains nil transaction")
		}

		// Apply transaction only to temporary state.
		minerFee, err := ApplyTransaction(workingState, tx)
		if err != nil {
			return 0, fmt.Errorf(
				"transaction %s failed: %w",
				tx.ID,
				err,
			)
		}

		// Protect against uint64 overflow.
		if totalMinerFees > ^uint64(0)-minerFee {
			return 0, fmt.Errorf("miner fee overflow")
		}

		totalMinerFees += minerFee
	}

	// Credit block subsidy + accumulated miner fees
	// to the temporary state.
	totalReward, err := CreditMinerReward(
		workingState,
		block.Height,
		minerAddress,
		totalMinerFees,
	)

	if err != nil {
		return 0, err
	}

	// Every transaction succeeded.
	// Commit the temporary state.
	if err := state.ReplaceWith(workingState); err != nil {
		return 0, err
	}

	return totalReward, nil
}

// TotalMinerFees calculates the miner portion of all
// transaction fees without modifying blockchain state.
func TotalMinerFees(
	txs []*transactions.Transaction,
) (uint64, error) {

	var total uint64

	for _, tx := range txs {
		if tx == nil {
			return 0, fmt.Errorf("nil transaction")
		}

		fee := transactions.MiningFee(tx.Amount)

		if total > ^uint64(0)-fee {
			return 0, fmt.Errorf("miner fee overflow")
		}

		total += fee
	}

	return total, nil
}
