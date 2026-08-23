package blockchain

import (
	"fmt"
	"sync"

	"github.com/sudharma-networks/sudharma/params"
)

type State struct {
	mu                 sync.RWMutex
	balances           map[string]uint64
	processedTx        map[string]bool
	accountNonces      map[string]uint64
	developmentAddress string
	issuedSupply       uint64
}

func NewState() *State {
	return &State{
		balances:           make(map[string]uint64),
		processedTx:        make(map[string]bool),
		accountNonces:      make(map[string]uint64),
		developmentAddress: params.DevelopmentTreasuryAddress,
	}
}

func (s *State) SetDevelopmentAddress(address string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Consensus rule:
	// Sudharma Network development fees always go to the
	// permanent network treasury address.
	if address != params.DevelopmentTreasuryAddress {
		return
	}

	s.developmentAddress = address
}

func (s *State) DevelopmentAddress() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.developmentAddress
}

func (s *State) Balance(address string) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.balances[address]
}

// Credit safely adds funds to an address.
func (s *State) Credit(address string, amount uint64) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	if amount == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.balances[address]

	if current > ^uint64(0)-amount {
		return fmt.Errorf("balance overflow")
	}

	s.balances[address] = current + amount

	return nil
}

func (s *State) Debit(address string, amount uint64) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	balance := s.balances[address]

	if balance < amount {
		return fmt.Errorf(
			"insufficient balance: have %d, need %d",
			balance,
			amount,
		)
	}

	s.balances[address] -= amount

	return nil
}

func (s *State) Transfer(
	from string,
	to string,
	amount uint64,
) error {
	if amount == 0 {
		return fmt.Errorf(
			"transfer amount must be greater than zero",
		)
	}

	if err := s.Debit(from, amount); err != nil {
		return err
	}

	if err := s.Credit(to, amount); err != nil {
		// Roll back debit if credit fails.
		_ = s.Credit(from, amount)
		return err
	}

	return nil
}

func (s *State) IsTransactionProcessed(txID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.processedTx[txID]
}

func (s *State) MarkTransactionProcessed(txID string) error {
	if txID == "" {
		return fmt.Errorf("transaction ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.processedTx[txID] {
		return fmt.Errorf(
			"transaction already processed: %s",
			txID,
		)
	}

	s.processedTx[txID] = true

	return nil
}

func (s *State) AccountNonce(address string) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.accountNonces[address]
}

func (s *State) ExpectedNonce(address string) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	current := s.accountNonces[address]

	if current == ^uint64(0) {
		return 0, fmt.Errorf("account nonce overflow")
	}

	return current + 1, nil
}

func (s *State) SetAccountNonce(
	address string,
	nonce uint64,
) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.accountNonces[address] = nonce

	return nil
}

// IssuedSupply returns the total SUDH created by block subsidies.
func (s *State) IssuedSupply() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.issuedSupply
}

// MintSupply increases issued supply while enforcing MaxSupply.
func (s *State) MintSupply(amount uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if amount == 0 {
		return nil
	}

	if s.issuedSupply > params.MaxSupply {
		return fmt.Errorf("issued supply already exceeds max supply")
	}

	remaining := params.MaxSupply - s.issuedSupply

	if amount > remaining {
		return fmt.Errorf(
			"maximum Sudharma Network supply exceeded",
		)
	}

	s.issuedSupply += amount

	return nil
}

func (s *State) Clone() *State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := &State{
		balances:           make(map[string]uint64, len(s.balances)),
		processedTx:        make(map[string]bool, len(s.processedTx)),
		accountNonces:      make(map[string]uint64, len(s.accountNonces)),
		developmentAddress: s.developmentAddress,
		issuedSupply:       s.issuedSupply,
	}

	for address, balance := range s.balances {
		clone.balances[address] = balance
	}

	for txID, processed := range s.processedTx {
		clone.processedTx[txID] = processed
	}

	for address, nonce := range s.accountNonces {
		clone.accountNonces[address] = nonce
	}

	return clone
}

func (s *State) ReplaceWith(other *State) error {
	if other == nil {
		return fmt.Errorf("replacement state cannot be nil")
	}

	other.mu.RLock()

	newBalances := make(
		map[string]uint64,
		len(other.balances),
	)

	for address, balance := range other.balances {
		newBalances[address] = balance
	}

	newProcessedTx := make(
		map[string]bool,
		len(other.processedTx),
	)

	for txID, processed := range other.processedTx {
		newProcessedTx[txID] = processed
	}

	newAccountNonces := make(
		map[string]uint64,
		len(other.accountNonces),
	)

	for address, nonce := range other.accountNonces {
		newAccountNonces[address] = nonce
	}

	newDevelopmentAddress := other.developmentAddress
	newIssuedSupply := other.issuedSupply

	other.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.balances = newBalances
	s.processedTx = newProcessedTx
	s.accountNonces = newAccountNonces
	s.developmentAddress = newDevelopmentAddress
	s.issuedSupply = newIssuedSupply

	return nil
}
