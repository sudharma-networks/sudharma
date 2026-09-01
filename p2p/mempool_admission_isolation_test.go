package p2p

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestLocalSubmissionIgnoresUnrelatedStaleMempoolEntry(t *testing.T) {
	chain := blockchain.NewChain()
	state := blockchain.NewState()
	node, err := NewNode("submit-isolation", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := node.SetChain(chain); err != nil {
		t.Fatal(err)
	}
	if err := node.SetState(state); err != nil {
		t.Fatal(err)
	}

	fundedSender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	staleSender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Credit(fundedSender.Address, 10*params.CoinDecimals); err != nil {
		t.Fatal(err)
	}

	// A state-invalid transaction can become stale after a chain advance/reorg.
	// It must not force every unrelated sender submission to replay and fail on it.
	stale := transactions.NewTransaction(
		staleSender.Address,
		receiver.Address,
		params.MinTransferAmount,
		1,
	)
	if err := stale.Sign(staleSender); err != nil {
		t.Fatal(err)
	}
	if err := node.Mempool().AddTransaction(stale); err != nil {
		t.Fatalf("failed to seed stale mempool fixture: %v", err)
	}

	valid := transactions.NewTransaction(
		fundedSender.Address,
		receiver.Address,
		params.CoinDecimals,
		1,
	)
	if err := valid.Sign(fundedSender); err != nil {
		t.Fatal(err)
	}
	if _, err := node.SubmitTransaction(valid); err != nil {
		t.Fatalf("unrelated stale mempool entry blocked valid submission: %v", err)
	}
	if _, ok := node.MempoolTransaction(valid.ID); !ok {
		t.Fatal("valid transaction was not retained in mempool")
	}
}
