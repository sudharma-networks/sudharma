package blockchain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sudharma-networks/sudharma/params"
)

// stateDiskData is the JSON representation of Sudharma Network's
// confirmed blockchain state.
//
// We intentionally keep this separate from State because
// State contains a mutex which must never be serialized.
type stateDiskData struct {
	Version            uint32            `json:"version"`
	MonetaryPolicy     uint8             `json:"monetary_policy,omitempty"`
	Balances           map[string]uint64 `json:"balances"`
	ProcessedTx        map[string]bool   `json:"processed_transactions"`
	AccountNonces      map[string]uint64 `json:"account_nonces"`
	DevelopmentAddress string            `json:"development_address"`
	IssuedSupply       uint64            `json:"issued_supply"`
}

// SaveToFile saves the complete confirmed blockchain state.
func (s *State) SaveToFile(path string) error {
	if s == nil {
		return fmt.Errorf("state cannot be nil")
	}

	if path == "" {
		return fmt.Errorf("state file path cannot be empty")
	}

	s.mu.RLock()

	data := stateDiskData{
		Version:            2,
		MonetaryPolicy:     uint8(s.monetaryPolicy),
		Balances:           make(map[string]uint64, len(s.balances)),
		ProcessedTx:        make(map[string]bool, len(s.processedTx)),
		AccountNonces:      make(map[string]uint64, len(s.accountNonces)),
		DevelopmentAddress: s.developmentAddress,
		IssuedSupply:       s.issuedSupply,
	}

	for address, balance := range s.balances {
		data.Balances[address] = balance
	}

	for txID, processed := range s.processedTx {
		data.ProcessedTx[txID] = processed
	}

	for address, nonce := range s.accountNonces {
		data.AccountNonces[address] = nonce
	}

	s.mu.RUnlock()

	encoded, err := json.MarshalIndent(
		data,
		"",
		"  ",
	)
	if err != nil {
		return fmt.Errorf(
			"failed to encode blockchain state: %w",
			err,
		)
	}

	directory := filepath.Dir(path)

	if directory != "." && directory != "" {
		if err := os.MkdirAll(
			directory,
			0700,
		); err != nil {
			return fmt.Errorf(
				"failed to create state directory: %w",
				err,
			)
		}
	}

	// Write to a temporary file first.
	// This reduces the risk of corrupting the real state file
	// if the process stops during a write.
	tempPath := path + ".tmp"

	if err := os.WriteFile(
		tempPath,
		encoded,
		0600,
	); err != nil {
		return fmt.Errorf(
			"failed to write temporary state file: %w",
			err,
		)
	}

	// On Windows Rename cannot replace an existing file,
	// so remove the old state file first.
	if err := os.Remove(path); err != nil &&
		!os.IsNotExist(err) {

		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"failed to replace old state file: %w",
			err,
		)
	}

	if err := os.Rename(
		tempPath,
		path,
	); err != nil {

		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"failed to finalize state file: %w",
			err,
		)
	}

	return nil
}

// LoadStateFromFile loads Sudharma Network's confirmed blockchain state.
func LoadStateFromFile(path string) (*State, error) {
	if path == "" {
		return nil, fmt.Errorf(
			"state file path cannot be empty",
		)
	}

	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var data stateDiskData

	if err := json.Unmarshal(
		encoded,
		&data,
	); err != nil {

		return nil, fmt.Errorf(
			"failed to decode blockchain state: %w",
			err,
		)
	}

	if data.Version != 1 && data.Version != 2 {
		return nil, fmt.Errorf(
			"unsupported state version: %d",
			data.Version,
		)
	}

	policy := params.MonetaryPolicyPublicTestnet
	if data.Version >= 2 {
		policy = params.MonetaryPolicy(data.MonetaryPolicy)
		if err := params.ValidateMonetaryPolicy(policy); err != nil {
			return nil, fmt.Errorf("invalid monetary policy in state file: %w", err)
		}
	}

	// The treasury address is a consensus rule.
	// A state file must never be able to redirect
	// development fees to another address.
	if data.DevelopmentAddress !=
		params.DevelopmentTreasuryAddress {

		return nil, fmt.Errorf(
			"invalid development treasury address in state file",
		)
	}

	maxSupply := params.MaxSupplyFor(policy)
	if data.IssuedSupply > maxSupply {
		return nil, fmt.Errorf(
			"state issued supply exceeds monetary policy maximum supply",
		)
	}

	if data.Balances == nil {
		data.Balances =
			make(map[string]uint64)
	}

	if data.ProcessedTx == nil {
		data.ProcessedTx =
			make(map[string]bool)
	}

	if data.AccountNonces == nil {
		data.AccountNonces =
			make(map[string]uint64)
	}

	state := &State{
		balances:           data.Balances,
		processedTx:        data.ProcessedTx,
		accountNonces:      data.AccountNonces,
		developmentAddress: params.DevelopmentTreasuryAddress,
		issuedSupply:       data.IssuedSupply,
		monetaryPolicy:     policy,
	}

	return state, nil
}
