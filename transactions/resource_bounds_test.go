package transactions

import (
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestValidateResourceBoundsRejectsDust(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := NewTransaction(
		sender.Address,
		receiver.Address,
		params.MinTransferAmount-1,
		0,
	)
	if err := ValidateResourceBounds(tx); err == nil {
		t.Fatal("dust transfer was accepted")
	}
}

func TestValidateResourceBoundsRejectsOversizedReceiver(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := NewTransaction(
		sender.Address,
		strings.Repeat("a", wallet.AddressLength+1),
		params.MinTransferAmount,
		0,
	)
	if err := ValidateResourceBounds(tx); err == nil {
		t.Fatal("oversized receiver was accepted")
	}
}
