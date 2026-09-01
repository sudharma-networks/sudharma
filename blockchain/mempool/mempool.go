package mempool

import (
	"fmt"
	"sync"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
)

// Mempool stores pending Sudharma Network transactions
// that have not yet been included in a block.
type Mempool struct {
	mu           sync.RWMutex
	transactions map[string]*transactions.Transaction
}

// NewMempool creates an empty transaction pool.
func NewMempool() *Mempool {
	return &Mempool{
		transactions: make(map[string]*transactions.Transaction),
	}
}

// AddTransaction validates and adds a transaction to the mempool.
func (m *Mempool) AddTransaction(tx *transactions.Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	if err := transactions.ValidateResourceBounds(tx); err != nil {
		return fmt.Errorf("transaction rejected: %w", err)
	}

	if tx.ID == "" {
		return fmt.Errorf("transaction ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.transactions[tx.ID]; exists {
		return fmt.Errorf("transaction already exists: %s", tx.ID)
	}
	if len(m.transactions) >= params.MaxMempoolTransactions {
		return fmt.Errorf("mempool transaction capacity reached")
	}
	if m.totalEstimatedBytesLocked()+tx.EstimatedSerializedSize() > params.MaxMempoolBytes {
		return fmt.Errorf("mempool byte capacity reached")
	}

	m.transactions[tx.ID] = tx

	return nil
}

// GetTransaction returns a transaction by ID.
func (m *Mempool) GetTransaction(id string) (*transactions.Transaction, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tx, exists := m.transactions[id]
	return tx, exists
}

// RemoveTransaction removes a transaction from the mempool.
func (m *Mempool) RemoveTransaction(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.transactions, id)
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

// TotalEstimatedBytes returns the approximate serialized size of all pending
// transactions.
func (m *Mempool) TotalEstimatedBytes() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalEstimatedBytesLocked()
}

func (m *Mempool) totalEstimatedBytesLocked() int {
	total := 0
	for _, tx := range m.transactions {
		total += tx.EstimatedSerializedSize()
	}
	return total
}

// AtCapacity reports whether another average-sized transaction should be
// rejected before expensive replay validation.
func (m *Mempool) AtCapacity() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.transactions) >= params.MaxMempoolTransactions ||
		m.totalEstimatedBytesLocked() >= params.MaxMempoolBytes
}

// Clear removes all transactions from the mempool.
func (m *Mempool) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.transactions = make(map[string]*transactions.Transaction)
}
