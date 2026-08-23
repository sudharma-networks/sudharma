package blockchain

import (
	"fmt"
	"sort"

	"github.com/sudharma-networks/sudharma/transactions"
)

// ValidateMempoolTransaction checks whether a transaction
// can safely enter the mempool.
//
// It validates against:
//
//   - current confirmed blockchain state
//   - transactions already waiting in the mempool
//
// The real blockchain state is never modified.
func ValidateMempoolTransaction(
	state *State,
	pending []*transactions.Transaction,
	candidate *transactions.Transaction,
) error {

	if state == nil {
		return fmt.Errorf(
			"state cannot be nil",
		)
	}

	if candidate == nil {
		return fmt.Errorf(
			"candidate transaction cannot be nil",
		)
	}

	if candidate.ID == "" {
		return fmt.Errorf(
			"transaction ID cannot be empty",
		)
	}

	// Reject duplicate transaction already in mempool.
	for _, tx := range pending {
		if tx == nil {
			return fmt.Errorf(
				"mempool contains nil transaction",
			)
		}

		if tx.ID == candidate.ID {
			return fmt.Errorf(
				"transaction already exists in mempool: %s",
				candidate.ID,
			)
		}
	}

	// Never modify confirmed blockchain state.
	workingState := state.Clone()

	// Make pending execution deterministic.
	ordered := make(
		[]*transactions.Transaction,
		len(pending),
	)

	copy(
		ordered,
		pending,
	)

	sort.Slice(
		ordered,
		func(i, j int) bool {

			if ordered[i].From != ordered[j].From {
				return ordered[i].From <
					ordered[j].From
			}

			if ordered[i].Nonce != ordered[j].Nonce {
				return ordered[i].Nonce <
					ordered[j].Nonce
			}

			return ordered[i].ID <
				ordered[j].ID
		},
	)

	// Apply already-pending transactions to temporary state.
	for _, tx := range ordered {
		if _, err := ApplyTransaction(
			workingState,
			tx,
		); err != nil {

			return fmt.Errorf(
				"existing mempool transaction %s is invalid: %w",
				tx.ID,
				err,
			)
		}
	}

	// Candidate must be valid after all pending transactions.
	if _, err := ApplyTransaction(
		workingState,
		candidate,
	); err != nil {

		return fmt.Errorf(
			"transaction rejected by mempool: %w",
			err,
		)
	}

	return nil
}
