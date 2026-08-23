package mempool

import (
	"fmt"
	"sync"

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

	if tx.ID == "" {
		return fmt.Errorf("transaction ID cannot be empty")
	}

	if tx.From == "" {
		return fmt.Errorf("transaction sender cannot be empty")
	}

	if tx.To == "" {
		return fmt.Errorf("transaction receiver cannot be empty")
	}

	if tx.Amount == 0 {
		return fmt.Errorf("transaction amount must be greater than zero")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.transactions[tx.ID]; exists {
		return fmt.Errorf("transaction already exists: %s", tx.ID)
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

// Clear removes all transactions from the mempool.
func (m *Mempool) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.transactions = make(map[string]*transactions.Transaction)
}
