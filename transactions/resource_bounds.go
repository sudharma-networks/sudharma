package transactions

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/wallet"
)

// EstimatedSerializedSize approximates the in-memory/relay cost of a transaction.
func (tx *Transaction) EstimatedSerializedSize() int {
	if tx == nil {
		return 0
	}
	return len(tx.ID) +
		len(tx.From) +
		len(tx.To) +
		len(tx.PublicKey) +
		len(tx.Signature) +
		24
}

// ValidateResourceBounds checks consensus-visible transaction size and address
// rules before expensive mempool replay or state application.
func ValidateResourceBounds(tx *Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}
	if len(tx.ID) == 0 || len(tx.ID) > params.MaxTransactionIDLength {
		return fmt.Errorf("invalid transaction ID length")
	}
	if err := wallet.ValidateAddress(tx.From); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if err := wallet.ValidateAddress(tx.To); err != nil {
		return fmt.Errorf("invalid receiver address: %w", err)
	}
	if len(tx.PublicKey) > params.MaxTransactionPublicKeySize {
		return fmt.Errorf("transaction public key exceeds maximum size")
	}
	if len(tx.Signature) > params.MaxTransactionSignatureSize {
		return fmt.Errorf("transaction signature exceeds maximum size")
	}
	if tx.EstimatedSerializedSize() > params.MaxBlockTransactionBytes {
		return fmt.Errorf("transaction exceeds maximum serialized size")
	}
	if tx.Amount < params.MinTransferAmount {
		return fmt.Errorf(
			"transfer amount below dust minimum %d",
			params.MinTransferAmount,
		)
	}
	return nil
}
