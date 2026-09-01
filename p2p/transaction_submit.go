package p2p

import (
	"fmt"
	"sync"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/transactions"
)

// Transaction admission can arrive concurrently through RPC and multiple peer
// read loops. Validation must see a stable sender-pending snapshot through the
// subsequent mempool insert, or same-nonce conflicts could both validate.
var transactionAdmissionLocks sync.Map // map[*Node]*sync.Mutex

func transactionAdmissionLock(node *Node) *sync.Mutex {
	lock, _ := transactionAdmissionLocks.LoadOrStore(node, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// admitTransaction is the single state-aware mempool admission path for both
// local RPC and peer-originated transfers. Cheap resource/capacity checks happen
// before signature/state work, and state replay is limited to the candidate
// sender's bounded pending queue.
func (n *Node) admitTransaction(tx *transactions.Transaction) error {
	if n == nil {
		return fmt.Errorf("node is unavailable")
	}
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}
	if err := transactions.ValidateResourceBounds(tx); err != nil {
		return fmt.Errorf("transaction rejected by resource policy: %w", err)
	}
	if err := n.mempool.CheckAdmission(tx); err != nil {
		return fmt.Errorf("transaction rejected by mempool capacity/index policy: %w", err)
	}
	if !tx.VerifyForNetwork(n.ActiveNetwork()) {
		return fmt.Errorf("transaction signature or identity is invalid")
	}

	admissionLock := transactionAdmissionLock(n)
	admissionLock.Lock()
	defer admissionLock.Unlock()

	state := n.State()
	if state == nil {
		return fmt.Errorf("blockchain state is unavailable")
	}
	if state.IsTransactionProcessed(tx.ID) {
		return fmt.Errorf("transaction already confirmed: %s", tx.ID)
	}

	// Repeat index-backed checks inside the serialized admission section so a
	// concurrent accepted transaction cannot invalidate the earlier preflight.
	if err := n.mempool.CheckAdmission(tx); err != nil {
		return fmt.Errorf("transaction rejected by mempool capacity/index policy: %w", err)
	}
	pending := n.mempool.TransactionsForSender(tx.From)
	if err := blockchain.ValidateMempoolTransactionFor(
		state,
		pending,
		tx,
		n.ActiveNetwork(),
	); err != nil {
		return fmt.Errorf("transaction rejected by mempool validation: %w", err)
	}
	if err := n.mempool.AddTransaction(tx); err != nil {
		return fmt.Errorf("failed to add transaction to mempool: %w", err)
	}
	return nil
}

// SubmitTransaction validates a locally submitted transaction using the same
// admission rules used for peer-originated transactions, stores it in the local
// mempool, and relays it to connected peers.
func (n *Node) SubmitTransaction(tx *transactions.Transaction) (int, error) {
	if err := n.admitTransaction(tx); err != nil {
		return 0, err
	}

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
