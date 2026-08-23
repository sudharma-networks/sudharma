package p2p

import (
	"fmt"
	"sync"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/transactions"
)

// Local submissions can arrive concurrently through RPC. Validation must see
// a stable pending-set snapshot through the subsequent mempool insert, or two
// same-nonce transactions could both validate before either is stored.
var localSubmissionLocks sync.Map // map[*Node]*sync.Mutex

func localSubmissionLock(node *Node) *sync.Mutex {
	lock, _ := localSubmissionLocks.LoadOrStore(node, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// SubmitTransaction validates a locally submitted transaction using the same
// state and mempool rules used for peer-originated transactions, stores it in
// the local mempool, and relays it to connected peers.
func (n *Node) SubmitTransaction(tx *transactions.Transaction) (int, error) {
	if n == nil {
		return 0, fmt.Errorf("node is unavailable")
	}
	if tx == nil {
		return 0, fmt.Errorf("transaction cannot be nil")
	}
	if !tx.Verify() {
		return 0, fmt.Errorf("transaction signature or identity is invalid")
	}

	admissionLock := localSubmissionLock(n)
	admissionLock.Lock()

	state := n.State()
	if state == nil {
		admissionLock.Unlock()
		return 0, fmt.Errorf("blockchain state is unavailable")
	}
	if state.IsTransactionProcessed(tx.ID) {
		admissionLock.Unlock()
		return 0, fmt.Errorf("transaction already confirmed: %s", tx.ID)
	}
	if _, exists := n.mempool.GetTransaction(tx.ID); exists {
		admissionLock.Unlock()
		return 0, fmt.Errorf("transaction already exists in mempool: %s", tx.ID)
	}

	pending := n.mempool.AllTransactions()
	if err := blockchain.ValidateMempoolTransaction(state, pending, tx); err != nil {
		admissionLock.Unlock()
		return 0, fmt.Errorf("transaction rejected by mempool validation: %w", err)
	}
	if err := n.mempool.AddTransaction(tx); err != nil {
		admissionLock.Unlock()
		return 0, fmt.Errorf("failed to add transaction to mempool: %w", err)
	}
	admissionLock.Unlock()

	// Relaying can involve network I/O and does not participate in admission
	// atomicity, so it intentionally happens outside the admission lock.
	sent, err := n.relayTransaction(tx, "")
	if err != nil {
		// The transaction is already accepted locally at this point. Keep it in
		// the mempool and surface the relay problem distinctly to the caller.
		return sent, fmt.Errorf("transaction accepted locally but relay failed: %w", err)
	}
	return sent, nil
}
