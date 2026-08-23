package wallet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncryptedWalletSaveAndLoad(t *testing.T) {
	original, err :=
		NewWallet()

	if err != nil {
		t.Fatal(err)
	}

	path :=
		filepath.Join(
			t.TempDir(),
			"encrypted-wallet.json",
		)

	password :=
		"correct-horse-battery-staple"

	if err :=
		original.SaveEncrypted(
			path,
			password,
		); err != nil {

		t.Fatal(err)
	}

	loaded, err :=
		LoadEncrypted(
			path,
			password,
		)

	if err != nil {
		t.Fatal(err)
	}

	if original.Address !=
		loaded.Address {

		t.Fatalf(
			"address changed after encrypted reload: expected %s, got %s",
			original.Address,
			loaded.Address,
		)
	}

	message :=
		[]byte(
			"Sudharma Network encrypted wallet test",
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
			"reloaded encrypted wallet signature failed",
		)
	}
}

func TestWrongWalletPasswordRejected(t *testing.T) {
	w, err :=
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
		w.SaveEncrypted(
			path,
			"this-is-the-correct-password",
		); err != nil {

		t.Fatal(err)
	}

	if _, err :=
		LoadEncrypted(
			path,
			"this-is-the-wrong-password",
		); err == nil {

		t.Fatal(
			"wrong wallet password was accepted",
		)
	}
}

func TestEncryptedWalletDoesNotContainRawPrivateKey(t *testing.T) {
	w, err :=
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
		w.SaveEncrypted(
			path,
			"very-long-development-password",
		); err != nil {

		t.Fatal(err)
	}

	data, err :=
		os.ReadFile(path)

	if err != nil {
		t.Fatal(err)
	}

	rawPrivateKey :=
		w.PrivateKey.D.Text(16)

	if strings.Contains(
		string(data),
		rawPrivateKey,
	) {
		t.Fatal(
			"encrypted wallet exposes raw private key",
		)
	}
}

func TestShortWalletPasswordRejected(t *testing.T) {
	w, err :=
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
		w.SaveEncrypted(
			path,
			"short",
		); err == nil {

		t.Fatal(
			"short wallet password was accepted",
		)
	}
}
