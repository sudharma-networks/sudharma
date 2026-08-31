package compatibility

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

type androidWalletGoldenVector struct {
	Version          int    `json:"version"`
	PrivateScalarHex string `json:"private_scalar_hex"`
	PublicKeyHex     string `json:"public_key_hex"`
	Address          string `json:"address"`
	Recipient        string `json:"recipient"`
	Amount           uint64 `json:"amount"`
	Fee              uint64 `json:"fee"`
	Nonce            uint64 `json:"nonce"`
	TransactionID    string `json:"transaction_id"`
	SignatureHex     string `json:"signature_hex"`
}

func TestAndroidWalletGoldenVectorMatchesGoProtocol(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/android_wallet_protocol_v1.json")
	if err != nil {
		t.Fatalf("read shared Android wallet golden vector: %v", err)
	}

	var vector androidWalletGoldenVector
	if err := json.Unmarshal(encoded, &vector); err != nil {
		t.Fatalf("decode shared Android wallet golden vector: %v", err)
	}
	if vector.Version != 1 {
		t.Fatalf("unsupported golden vector version: %d", vector.Version)
	}

	privateScalar, err := hex.DecodeString(vector.PrivateScalarHex)
	if err != nil {
		t.Fatalf("decode private scalar: %v", err)
	}
	x, y := elliptic.P256().ScalarBaseMult(privateScalar)
	publicKey := elliptic.Marshal(elliptic.P256(), x, y)
	if got := hex.EncodeToString(publicKey); got != vector.PublicKeyHex {
		t.Fatalf("public key mismatch: got %s, want %s", got, vector.PublicKeyHex)
	}
	if got := wallet.AddressFromPublicKey(publicKey); got != vector.Address {
		t.Fatalf("address mismatch: got %s, want %s", got, vector.Address)
	}

	tx := transactions.NewTransaction(
		vector.Address,
		vector.Recipient,
		vector.Amount,
		vector.Nonce,
	)
	if tx.Fee != vector.Fee {
		t.Fatalf("fee mismatch: got %d, want %d", tx.Fee, vector.Fee)
	}
	if tx.ID != vector.TransactionID {
		t.Fatalf("transaction ID mismatch: got %s, want %s", tx.ID, vector.TransactionID)
	}

	signature, err := hex.DecodeString(vector.SignatureHex)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	if len(signature) != 64 {
		t.Fatalf("signature length mismatch: got %d, want 64", len(signature))
	}
	key := &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}
	if !wallet.VerifySignature(
		elliptic.Marshal(key.Curve, key.X, key.Y),
		[]byte(vector.TransactionID),
		signature,
	) {
		t.Fatal("shared Android signature is not accepted by Go verification")
	}
}
