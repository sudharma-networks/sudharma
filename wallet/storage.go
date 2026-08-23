package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

type walletFile struct {
	PrivateKey string `json:"private_key"`
}

// SaveToFile saves the wallet private key.
//
// IMPORTANT:
// This development wallet format is NOT encrypted yet.
// The file must be protected and must never be committed
// to Git or shared with anyone.
func (w *Wallet) SaveToFile(path string) error {
	if w == nil {
		return fmt.Errorf("wallet cannot be nil")
	}

	if w.PrivateKey == nil {
		return fmt.Errorf("wallet private key cannot be nil")
	}

	if path == "" {
		return fmt.Errorf("wallet path cannot be empty")
	}

	data, err := json.MarshalIndent(
		walletFile{
			PrivateKey: hex.EncodeToString(
				w.PrivateKey.D.Bytes(),
			),
		},
		"",
		"  ",
	)

	if err != nil {
		return fmt.Errorf(
			"failed to encode wallet: %w",
			err,
		)
	}

	directory := filepath.Dir(path)

	if directory != "." {
		if err := os.MkdirAll(
			directory,
			0700,
		); err != nil {

			return fmt.Errorf(
				"failed to create wallet directory: %w",
				err,
			)
		}
	}

	if err := os.WriteFile(
		path,
		data,
		0600,
	); err != nil {

		return fmt.Errorf(
			"failed to save wallet: %w",
			err,
		)
	}

	return nil
}

// LoadFromFile restores a Sudharma Network wallet
// from a previously saved private key.
func LoadFromFile(path string) (*Wallet, error) {
	if path == "" {
		return nil,
			fmt.Errorf("wallet path cannot be empty")
	}

	data, err := os.ReadFile(path)

	if err != nil {
		return nil,
			fmt.Errorf(
				"failed to read wallet: %w",
				err,
			)
	}

	var stored walletFile

	if err := json.Unmarshal(
		data,
		&stored,
	); err != nil {

		return nil,
			fmt.Errorf(
				"invalid wallet file: %w",
				err,
			)
	}

	privateBytes, err :=
		hex.DecodeString(
			stored.PrivateKey,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"invalid private key encoding: %w",
				err,
			)
	}

	if len(privateBytes) == 0 {
		return nil,
			fmt.Errorf(
				"private key cannot be empty",
			)
	}

	curve := elliptic.P256()

	d := new(big.Int).SetBytes(
		privateBytes,
	)

	if d.Sign() <= 0 ||
		d.Cmp(curve.Params().N) >= 0 {

		return nil,
			fmt.Errorf(
				"invalid private key value",
			)
	}

	x, y :=
		curve.ScalarBaseMult(
			d.Bytes(),
		)

	privateKey :=
		&ecdsa.PrivateKey{
			PublicKey: ecdsa.PublicKey{
				Curve: curve,
				X:     x,
				Y:     y,
			},
			D: d,
		}

	publicKey :=
		elliptic.Marshal(
			curve,
			x,
			y,
		)

	address :=
		AddressFromPublicKey(
			publicKey,
		)

	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
	}, nil
}
