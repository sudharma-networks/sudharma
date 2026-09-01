package transactions

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/wallet"
)

const (
	// SignatureDomainLegacy signs only the transaction ID. Existing public-testnet
	// transactions remain valid under this domain until operators migrate.
	SignatureDomainLegacy = 1

	// SignatureDomainNetworkBound binds the signature to an immutable network ID.
	SignatureDomainNetworkBound = 2
)

// SigningMessage returns the exact bytes wallets and nodes must sign for a domain.
func SigningMessage(
	domain int,
	network params.NetworkID,
	txID string,
) ([]byte, error) {
	switch domain {
	case SignatureDomainLegacy:
		return []byte(txID), nil
	case SignatureDomainNetworkBound:
		if network == "" {
			return nil, fmt.Errorf("network cannot be empty")
		}
		if txID == "" {
			return nil, fmt.Errorf("transaction ID cannot be empty")
		}
		return []byte(
			fmt.Sprintf("sudharma-tx-v2|%s|%s", network, txID),
		), nil
	default:
		return nil, fmt.Errorf("unknown signature domain %d", domain)
	}
}

func verifySignatureDomain(
	tx *Transaction,
	network params.NetworkID,
	domain int,
) bool {
	if tx == nil {
		return false
	}

	message, err := SigningMessage(domain, network, tx.ID)
	if err != nil {
		return false
	}

	return wallet.VerifySignature(
		tx.PublicKey,
		message,
		tx.Signature,
	)
}
