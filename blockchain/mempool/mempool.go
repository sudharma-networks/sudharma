package mempool

import (
	"fmt"
	"sort"
	"sync"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
)

type entryMetadata struct {
	sender string
	nonce  uint64
	size   int
}

// Mempool stores pending Sudharma Network transactions that have not yet been
// included in a block. Resource accounting and sender/nonce indexes are kept
// alongside the ID map so admission checks do not rescan the full pool.
type Mempool struct {
	mu                  sync.RWMutex
	transactions        map[string]*transactions.Transaction
	metadata            map[string]entryMetadata
	totalEstimatedBytes int
	senderNonces        map[string]map[uint64]string
}

// NewMempool creates an empty transaction pool.
func NewMempool() *Mempool {
	return &Mempool{
		transactions: make(map[string]*transactions.Transaction),
		metadata:     make(map[string]entryMetadata),
		senderNonces: make(map[string]map[uint64]string),
	}
}

// CheckAdmission performs cheap, index-backed resource/duplicate checks before
// expensive state replay or signature verification. AddTransaction repeats the
// same checks under the write lock so concurrent callers remain fail-closed.
func (m *Mempool) CheckAdmission(tx *transactions.Transaction) error {
	if m == nil {
		return fmt.Errorf("mempool cannot be nil")
	}
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}
	if err := transactions.ValidateResourceBounds(tx); err != nil {
		return fmt.Errorf("transaction rejected: %w", err)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.checkAdmissionLocked(tx)
}

// AddTransaction validates and adds a transaction to the mempool.
func (m *Mempool) AddTransaction(tx *transactions.Transaction) error {
	if m == nil {
		return fmt.Errorf("mempool cannot be nil")
	}
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}
	if err := transactions.ValidateResourceBounds(tx); err != nil {
		return fmt.Errorf("transaction rejected: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.checkAdmissionLocked(tx); err != nil {
		return err
	}
	if m.transactions == nil {
		m.transactions = make(map[string]*transactions.Transaction)
	}
	if m.metadata == nil {
		m.metadata = make(map[string]entryMetadata)
	}
	if m.senderNonces == nil {
		m.senderNonces = make(map[string]map[uint64]string)
	}
	if m.senderNonces[tx.From] == nil {
		m.senderNonces[tx.From] = make(map[uint64]string)
	}

	meta := entryMetadata{
		sender: tx.From,
		nonce:  tx.Nonce,
		size:   tx.EstimatedSerializedSize(),
	}
	m.transactions[tx.ID] = tx
	m.metadata[tx.ID] = meta
	m.senderNonces[meta.sender][meta.nonce] = tx.ID
	m.totalEstimatedBytes += meta.size
	return nil
}

func (m *Mempool) checkAdmissionLocked(tx *transactions.Transaction) error {
	if tx.ID == "" {
		return fmt.Errorf("transaction ID cannot be empty")
	}
	if _, exists := m.transactions[tx.ID]; exists {
		return fmt.Errorf("transaction already exists: %s", tx.ID)
	}
	if len(m.transactions) >= params.MaxMempoolTransactions {
		return fmt.Errorf("mempool transaction capacity reached")
	}
	if m.totalEstimatedBytes+tx.EstimatedSerializedSize() > params.MaxMempoolBytes {
		return fmt.Errorf("mempool byte capacity reached")
	}

	senderNonces := m.senderNonces[tx.From]
	if len(senderNonces) >= params.MaxMempoolTransactionsPerSender {
		return fmt.Errorf("mempool sender transaction capacity reached")
	}
	if existingID, exists := senderNonces[tx.Nonce]; exists {
		return fmt.Errorf(
			"sender nonce already pending: sender=%s nonce=%d transaction=%s",
			tx.From,
			tx.Nonce,
			existingID,
		)
	}
	return nil
}

// GetTransaction returns a transaction by ID.
func (m *Mempool) GetTransaction(id string) (*transactions.Transaction, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tx, exists := m.transactions[id]
	return tx, exists
}

// TransactionsForSender returns the bounded pending chain for one sender in
// deterministic nonce order. Live admission uses this instead of copying and
// replaying every transaction in the global mempool.
func (m *Mempool) TransactionsForSender(sender string) []*transactions.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nonces := m.senderNonces[sender]
	result := make([]*transactions.Transaction, 0, len(nonces))
	for _, id := range nonces {
		if tx, ok := m.transactions[id]; ok && tx != nil {
			result = append(result, tx)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Nonce != result[j].Nonce {
			return result[i].Nonce < result[j].Nonce
		}
		return result[i].ID < result[j].ID
	})
	return result
}

// RemoveTransaction removes a transaction from the mempool and updates the
// cached resource/sender indexes atomically. Metadata captured at insertion is
// used so an accidentally mutated transaction pointer cannot corrupt accounting.
func (m *Mempool) RemoveTransaction(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.transactions[id]; !exists {
		return
	}
	delete(m.transactions, id)

	meta, ok := m.metadata[id]
	if !ok {
		// Every supported insertion path records metadata. If an impossible
		// inconsistent zero-value/manual state is observed, fail closed by
		// removing the ID while leaving accounting untouched rather than guessing.
		return
	}
	delete(m.metadata, id)

	if meta.size >= m.totalEstimatedBytes {
		m.totalEstimatedBytes = 0
	} else {
		m.totalEstimatedBytes -= meta.size
	}
	if nonces := m.senderNonces[meta.sender]; nonces != nil {
		if indexedID, exists := nonces[meta.nonce]; exists && indexedID == id {
			delete(nonces, meta.nonce)
		}
		if len(nonces) == 0 {
			delete(m.senderNonces, meta.sender)
		}
	}
}

// AllTransactions returns all pending transactions.
func (m *Mempool) AllTransactions() []*transactions.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*transactions.Transaction, 0, len(m.transactions))
	for _, tx := range m.transactions {
		result = append(result, tx)
	}
	return result
}

// Count returns the number of pending transactions.
func (m *Mempool) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.transactions)
}

// CountForSender returns the number of pending transactions indexed for one
// sender. It is bounded by MaxMempoolTransactionsPerSender.
func (m *Mempool) CountForSender(sender string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.senderNonces[sender])
}

// TotalEstimatedBytes returns cached resource usage in O(1).
func (m *Mempool) TotalEstimatedBytes() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalEstimatedBytes
}

// AtCapacity reports whether the global pool cannot accept another transaction
// before transaction-specific size/sender checks are considered.
func (m *Mempool) AtCapacity() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.transactions) >= params.MaxMempoolTransactions ||
		m.totalEstimatedBytes >= params.MaxMempoolBytes
}

// Clear removes all transactions and resets cached indexes/accounting.
func (m *Mempool) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions = make(map[string]*transactions.Transaction)
	m.metadata = make(map[string]entryMetadata)
	m.senderNonces = make(map[string]map[uint64]string)
	m.totalEstimatedBytes = 0
}
