package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
)

// DecodeTransactionForNetwork decodes a transaction message and verifies its
// signature against the explicit active network domain. DecodeTransaction in
// message.go remains the public-testnet compatibility wrapper for older callers.
func DecodeTransactionForNetwork(
	message *Message,
	network params.NetworkID,
) (*transactions.Transaction, error) {
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}
	if message.Type != MessageTransaction {
		return nil, fmt.Errorf("message is not a transaction")
	}
	var payload TransactionMessage
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid transaction message: %w", err)
	}
	if payload.Transaction == nil {
		return nil, fmt.Errorf("transaction payload is nil")
	}
	if err := transactions.ValidateResourceBounds(payload.Transaction); err != nil {
		return nil, fmt.Errorf("transaction resource bounds failed: %w", err)
	}
	if !payload.Transaction.VerifyForNetwork(network) {
		return nil, fmt.Errorf("transaction signature verification failed")
	}
	return payload.Transaction, nil
}
