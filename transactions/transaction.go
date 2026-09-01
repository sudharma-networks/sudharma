package transactions

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/wallet"
)

const (
	TotalFeeBasisPoints       uint64 = 10
	DevelopmentFeeBasisPoints uint64 = 1
	MiningFeeBasisPoints      uint64 = 9
	feeBasisPointsDivisor     uint64 = 10000
)

type Transaction struct {
	ID        string
	From      string
	To        string
	Amount    uint64
	Fee       uint64
	Nonce     uint64
	PublicKey []byte
	Signature []byte
}

// NewTransaction creates a Sudharma Network transaction.
//
// nonce uniquely identifies transactions from the same sender.
// Later the wallet/node will assign this automatically.
func NewTransaction(
	from string,
	to string,
	amount uint64,
	nonce ...uint64,
) *Transaction {

	var txNonce uint64

	if len(nonce) > 0 {
		txNonce = nonce[0]
	}

	tx := &Transaction{
		From:   from,
		To:     to,
		Amount: amount,
		Nonce:  txNonce,
	}

	tx.Fee = CalculateFee(amount)
	tx.ID = tx.calculateID()

	return tx
}

// basisPointsFee returns floor(amount*basisPoints/10000) without first
// multiplying the full amount. Splitting amount into quotient/remainder keeps
// the arithmetic safe for the complete uint64 transaction-amount domain.
func basisPointsFee(amount, basisPoints uint64) uint64 {
	whole := amount / feeBasisPointsDivisor
	remainder := amount % feeBasisPointsDivisor

	return whole*basisPoints + (remainder*basisPoints)/feeBasisPointsDivisor
}

// CalculateFee calculates the total 0.10% transaction fee.
func CalculateFee(amount uint64) uint64 {
	return basisPointsFee(amount, TotalFeeBasisPoints)
}

// DevelopmentFee returns the 0.01% development portion.
func DevelopmentFee(amount uint64) uint64 {
	return basisPointsFee(amount, DevelopmentFeeBasisPoints)
}

// MiningFee returns the miner portion of the charged fee.
//
// The miner receives the exact remainder after the development allocation so
// integer rounding can never create or destroy fee atoms. For amounts that are
// exactly representable at the configured basis-point precision this remains
// exactly 0.09%.
func MiningFee(amount uint64) uint64 {
	total := CalculateFee(amount)
	development := DevelopmentFee(amount)
	return total - development
}

// calculateID creates the deterministic transaction ID.
//
// The nonce is included so two otherwise identical payments
// can still have different transaction IDs.
func (tx *Transaction) calculateID() string {
	data := fmt.Sprintf(
		"%s|%s|%d|%d|%d",
		tx.From,
		tx.To,
		tx.Amount,
		tx.Fee,
		tx.Nonce,
	)

	hash := sha256.Sum256([]byte(data))

	return hex.EncodeToString(hash[:])
}

// Sign signs the transaction for the default public-testnet network using the
// current network-bound signature domain.
func (tx *Transaction) Sign(w *wallet.Wallet) error {
	return tx.SignForNetwork(w, params.DefaultNetwork)
}

// SignForNetwork signs the transaction using the network-bound signature domain.
func (tx *Transaction) SignForNetwork(
	w *wallet.Wallet,
	network params.NetworkID,
) error {
	if w == nil {
		return fmt.Errorf("wallet cannot be nil")
	}

	if tx.From != w.Address {
		return fmt.Errorf(
			"wallet address does not match transaction sender",
		)
	}

	tx.PublicKey = append(
		[]byte(nil),
		w.PublicKey...,
	)

	message, err := SigningMessage(
		SignatureDomainNetworkBound,
		network,
		tx.ID,
	)
	if err != nil {
		return err
	}

	signature, err := w.Sign(message)
	if err != nil {
		return err
	}

	tx.Signature = signature

	return nil
}

// Verify independently verifies the transaction on the default network.
func (tx *Transaction) Verify() bool {
	return tx.VerifyForNetwork(params.DefaultNetwork)
}

// VerifyForNetwork verifies the transaction signature for an active network.
//
// Public testnet accepts both the current network-bound domain and the legacy
// transaction-ID-only domain so already-signed transactions are not silently
// reinterpreted. Mainnet and other networks require the network-bound domain.
func (tx *Transaction) VerifyForNetwork(network params.NetworkID) bool {
	if len(tx.PublicKey) == 0 {
		return false
	}

	if len(tx.Signature) == 0 {
		return false
	}

	expectedAddress :=
		wallet.AddressFromPublicKey(tx.PublicKey)

	if expectedAddress != tx.From {
		return false
	}

	expectedID := tx.calculateID()

	if expectedID != tx.ID {
		return false
	}

	if verifySignatureDomain(
		tx,
		network,
		SignatureDomainNetworkBound,
	) {
		return true
	}

	if network == params.NetworkPublicTestnet {
		return verifySignatureDomain(
			tx,
			network,
			SignatureDomainLegacy,
		)
	}

	return false
}
