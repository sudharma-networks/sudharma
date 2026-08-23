package wallet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalletSaveAndLoad(t *testing.T) {
	original, err :=
		NewWallet()

	if err != nil {
		t.Fatal(err)
	}

	path :=
		filepath.Join(
			t.TempDir(),
			"wallet.json",
		)

	if err :=
		original.SaveToFile(path); err != nil {

		t.Fatalf(
			"failed to save wallet: %v",
			err,
		)
	}

	loaded, err :=
		LoadFromFile(path)

	if err != nil {
		t.Fatalf(
			"failed to load wallet: %v",
			err,
		)
	}

	if original.Address !=
		loaded.Address {

		t.Fatalf(
			"address changed after reload: expected %s, got %s",
			original.Address,
			loaded.Address,
		)
	}

	message :=
		[]byte(
			"Sudharma Network wallet persistence test",
		)

	signature, err :=
		loaded.Sign(message)

	if err != nil {
		t.Fatal(err)
	}

	if !original.Verify(
		message,
		signature,
	) {
		t.Fatal(
			"loaded private key produced invalid signature",
		)
	}
}

func TestCorruptedWalletRejected(t *testing.T) {
	path :=
		filepath.Join(
			t.TempDir(),
			"bad-wallet.json",
		)

	if err :=
		os.WriteFile(
			path,
			[]byte(
				`{"private_key":"not-hex"}`,
			),
			0600,
		); err != nil {

		t.Fatal(err)
	}

	if _, err :=
		LoadFromFile(path); err == nil {

		t.Fatal(
			"corrupted wallet was accepted",
		)
	}
}

func TestZeroPrivateKeyRejected(t *testing.T) {
	path :=
		filepath.Join(
			t.TempDir(),
			"zero-wallet.json",
		)

	if err :=
		os.WriteFile(
			path,
			[]byte(
				`{"private_key":"00"}`,
			),
			0600,
		); err != nil {

		t.Fatal(err)
	}

	if _, err :=
		LoadFromFile(path); err == nil {

		t.Fatal(
			"zero private key was accepted",
		)
	}
}
