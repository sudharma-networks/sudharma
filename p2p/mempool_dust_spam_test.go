package p2p

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestSequentialDustSpamIsRejectedWithoutChangingMempoolOrNonce(t *testing.T) {
	chain := blockchain.NewChain()
	state := blockchain.NewState()
	node, err := NewNode("dust-spam", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := node.SetChain(chain); err != nil {
		t.Fatal(err)
	}
	if err := node.SetState(state); err != nil {
		t.Fatal(err)
	}

	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Credit(sender.Address, params.CoinDecimals); err != nil {
		t.Fatal(err)
	}

	for nonce := uint64(1); nonce <= 32; nonce++ {
		tx := transactions.NewTransaction(
			sender.Address,
			receiver.Address,
			params.MinTransferAmount-1,
			nonce,
		)
		if err := tx.Sign(sender); err != nil {
			t.Fatal(err)
		}
		if _, err := node.SubmitTransaction(tx); err == nil {
			t.Fatalf("dust transaction nonce %d was accepted", nonce)
		}
	}

	if got := node.MempoolCount(); got != 0 {
		t.Fatalf("dust spam changed mempool count to %d", got)
	}
	if got := state.AccountNonce(sender.Address); got != 0 {
		t.Fatalf("dust spam changed confirmed nonce to %d", got)
	}
	if got := state.Balance(sender.Address); got != params.CoinDecimals {
		t.Fatalf("dust spam changed sender balance to %d", got)
	}
}
