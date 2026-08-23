package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"math/big"
)

func walletFromPrivateKeyBytes(
	privateBytes []byte,
) (*Wallet, error) {

	if len(privateBytes) == 0 {
		return nil,
			fmt.Errorf(
				"private key cannot be empty",
			)
	}

	curve :=
		elliptic.P256()

	d :=
		new(big.Int).SetBytes(
			privateBytes,
		)

	if d.Sign() <= 0 ||
		d.Cmp(curve.Params().N) >= 0 {

		return nil,
			fmt.Errorf(
				"invalid private key",
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
