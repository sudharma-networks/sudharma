package p2p

import (
	"fmt"
	"sort"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/transactions"
)

// ReorganizeWithMempoolRecovery adopts a better candidate chain
// and then rebuilds the local mempool.
//
// Transactions from the abandoned chain are reconsidered.
// They are returned to the mempool only when they remain valid
// under the newly adopted blockchain state.
func (n *Node) ReorganizeWithMempoolRecovery(
	candidate *blockchain.Chain,
) (bool, int, error) {

	if candidate == nil {
		return false, 0, fmt.Errorf(
			"candidate chain cannot be nil",
		)
	}

	current :=
		n.Chain()

	if current == nil {
		return false, 0, fmt.Errorf(
			"blockchain is not attached",
		)
	}

	state :=
		n.State()

	if state == nil {
		return false, 0, fmt.Errorf(
			"blockchain state is not attached",
		)
	}

	// Preserve the old blockchain before reorganizing.
	oldChain, err :=
		blockchain.CloneChain(
			current,
		)

	if err != nil {
		return false, 0, fmt.Errorf(
			"failed to clone old chain: %w",
			err,
		)
	}

	// Preserve transactions already waiting in the mempool.
	oldPending :=
		n.mempool.AllTransactions()

	adopted, err :=
		blockchain.ReorganizeToCandidate(
			current,
			state,
			candidate,
		)

	if err != nil {
		return false, 0, err
	}

	if !adopted {
		return false, 0, nil
	}

	// -------------------------------------------------
	// Build the set of transactions that may need to
	// return to the mempool.
	// -------------------------------------------------

	candidates :=
		make(
			map[string]*transactions.Transaction,
		)

	// Existing pending transactions should be reconsidered
	// because the confirmed state has changed.
	for _, tx := range oldPending {

		if tx == nil ||
			tx.ID == "" {

			continue
		}

		candidates[tx.ID] =
			tx
	}

	// Transactions confirmed only on the abandoned chain
	// become orphaned and should also be reconsidered.
	for height := uint64(1); height <= oldChain.Height(); height++ {

		block, ok :=
			oldChain.BlockByHeight(
				height,
			)

		if !ok ||
			block == nil {

			continue
		}

		for _, tx := range block.Transactions {

			if tx == nil ||
				tx.ID == "" {

				continue
			}

			// If this transaction is confirmed in the winning
			// chain, it must not return to the mempool.
			if state.IsTransactionProcessed(
				tx.ID,
			) {
				continue
			}

			candidates[tx.ID] =
				tx
		}
	}

	// -------------------------------------------------
	// Clear and rebuild the mempool from scratch.
	// -------------------------------------------------

	n.mempool.Clear()

	ordered :=
		make(
			[]*transactions.Transaction,
			0,
			len(candidates),
		)

	for _, tx := range candidates {

		ordered =
			append(
				ordered,
				tx,
			)
	}

	// Deterministic ordering is important when multiple
	// transactions belong to the same account.
	sort.Slice(
		ordered,
		func(i, j int) bool {

			if ordered[i].From !=
				ordered[j].From {

				return ordered[i].From <
					ordered[j].From
			}

			if ordered[i].Nonce !=
				ordered[j].Nonce {

				return ordered[i].Nonce <
					ordered[j].Nonce
			}

			return ordered[i].ID <
				ordered[j].ID
		},
	)

	recovered := 0

	for _, tx := range ordered {

		// Winning-chain confirmed transactions never belong
		// in the mempool.
		if state.IsTransactionProcessed(
			tx.ID,
		) {
			continue
		}

		pending :=
			n.mempool.AllTransactions()

		if err :=
			blockchain.ValidateMempoolTransaction(
				state,
				pending,
				tx,
			); err != nil {

			fmt.Printf(
				"[REORG] Orphaned transaction %s not recovered: %v\n",
				tx.ID,
				err,
			)

			continue
		}

		if err :=
			n.mempool.AddTransaction(
				tx,
			); err != nil {

			fmt.Printf(
				"[REORG] Failed returning transaction %s to mempool: %v\n",
				tx.ID,
				err,
			)

			continue
		}

		recovered++

		fmt.Printf(
			"[REORG] Returned orphaned transaction %s to mempool\n",
			tx.ID,
		)
	}

	n.RefreshChainStatus()

	return true,
		recovered,
		nil
}
