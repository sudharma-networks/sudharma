package blockchain

import (
	"fmt"
	"sort"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
)

// ValidateMempoolTransaction checks whether a transaction can safely enter the
// mempool on the default public-testnet network.
func ValidateMempoolTransaction(
	state *State,
	pending []*transactions.Transaction,
	candidate *transactions.Transaction,
) error {
	return ValidateMempoolTransactionFor(
		state,
		pending,
		candidate,
		params.DefaultNetwork,
	)
}

// ValidateMempoolTransactionFor validates a candidate against confirmed state
// plus only that sender's bounded pending chain. Cross-sender pending transfers
// are deliberately ignored: mempool admission does not permit spending
// unconfirmed incoming funds, which removes global replay coupling and keeps
// candidate validation cost bounded by MaxMempoolTransactionsPerSender.
//
// Global count/byte capacity and duplicate sender/nonce checks are enforced by
// mempool.CheckAdmission/AddTransaction before this function is called on live
// admission paths.
func ValidateMempoolTransactionFor(
	state *State,
	pending []*transactions.Transaction,
	candidate *transactions.Transaction,
	network params.NetworkID,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if candidate == nil {
		return fmt.Errorf("candidate transaction cannot be nil")
	}
	if candidate.ID == "" {
		return fmt.Errorf("transaction ID cannot be empty")
	}
	if err := transactions.ValidateResourceBounds(candidate); err != nil {
		return fmt.Errorf("transaction rejected by mempool: %w", err)
	}

	senderPending := make([]*transactions.Transaction, 0, params.MaxMempoolTransactionsPerSender)
	for _, tx := range pending {
		if tx == nil {
			return fmt.Errorf("mempool contains nil transaction")
		}
		if tx.ID == candidate.ID {
			return fmt.Errorf("transaction already exists in mempool: %s", candidate.ID)
		}
		if tx.From != candidate.From {
			continue
		}
		senderPending = append(senderPending, tx)
		if len(senderPending) >= params.MaxMempoolTransactionsPerSender {
			return fmt.Errorf("mempool sender transaction capacity reached")
		}
	}

	sort.Slice(senderPending, func(i, j int) bool {
		if senderPending[i].Nonce != senderPending[j].Nonce {
			return senderPending[i].Nonce < senderPending[j].Nonce
		}
		return senderPending[i].ID < senderPending[j].ID
	})

	// Never modify confirmed blockchain state.
	workingState := state.Clone()

	// Apply only this sender's already-pending transactions. Since one sender is
	// capped at a small fixed queue, this replay is bounded independent of the
	// global mempool size.
	for _, tx := range senderPending {
		if _, err := ApplyTransactionFor(workingState, tx, network); err != nil {
			return fmt.Errorf(
				"existing sender mempool transaction %s is invalid: %w",
				tx.ID,
				err,
			)
		}
	}

	if _, err := ApplyTransactionFor(workingState, candidate, network); err != nil {
		return fmt.Errorf("transaction rejected by mempool: %w", err)
	}
	return nil
}
