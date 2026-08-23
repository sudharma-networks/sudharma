package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte
	Address    string
}

func NewWallet() (*Wallet, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	publicKey := elliptic.Marshal(
		elliptic.P256(),
		privateKey.PublicKey.X,
		privateKey.PublicKey.Y,
	)

	address := AddressFromPublicKey(publicKey)

	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
	}, nil
}

// AddressFromPublicKey derives a Sudharma Network address from a public key.
func AddressFromPublicKey(publicKey []byte) string {
	hash := sha256.Sum256(publicKey)
	return hex.EncodeToString(hash[:20])
}

// Sign signs arbitrary data using the wallet private key.
func (w *Wallet) Sign(data []byte) ([]byte, error) {
	hash := sha256.Sum256(data)

	r, s, err := ecdsa.Sign(
		rand.Reader,
		w.PrivateKey,
		hash[:],
	)

	if err != nil {
		return nil, err
	}

	// Fixed-size 64-byte signature:
	// 32 bytes R + 32 bytes S.
	signature := make([]byte, 64)

	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])

	return signature, nil
}

// VerifySignature verifies a signature using only a public key.
func VerifySignature(
	publicKey []byte,
	data []byte,
	signature []byte,
) bool {

	if len(signature) != 64 {
		return false
	}

	x, y := elliptic.Unmarshal(
		elliptic.P256(),
		publicKey,
	)

	if x == nil || y == nil {
		return false
	}

	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	hash := sha256.Sum256(data)

	key := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	return ecdsa.Verify(
		&key,
		hash[:],
		r,
		s,
	)
}

// Verify verifies a signature using this wallet's public key.
func (w *Wallet) Verify(data []byte, signature []byte) bool {
	return VerifySignature(
		w.PublicKey,
		data,
		signature,
	)
}

func (w *Wallet) String() string {
	return fmt.Sprintf(
		"Sudharma Network Wallet\nAddress: %s",
		w.Address,
	)
}
