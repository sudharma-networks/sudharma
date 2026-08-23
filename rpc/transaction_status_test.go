package rpc

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sudharma-networks/sudharma/miner"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestTransactionLifecyclePendingThenConfirmed(t *testing.T) {
	server, node, chain, state := newTestServer(t)
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Credit(sender.Address, 10*params.CoinDecimals); err != nil {
		t.Fatal(err)
	}
	tx := transactions.NewTransaction(sender.Address, receiver.Address, params.CoinDecimals, 1)
	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}
	if _, err := node.SubmitTransaction(tx); err != nil {
		t.Fatal(err)
	}

	pending := request(t, server, http.MethodGet, "/v1/transactions/"+tx.ID, nil, "")
	if pending.Code != http.StatusOK {
		t.Fatalf("pending status returned %d: %s", pending.Code, pending.Body.String())
	}
	var pendingStatus transactionStatusResponse
	if err := json.Unmarshal(pending.Body.Bytes(), &pendingStatus); err != nil {
		t.Fatal(err)
	}
	if pendingStatus.Status != "pending" || pendingStatus.Confirmations != 0 {
		t.Fatalf("unexpected pending status: %+v", pendingStatus)
	}

	minerWallet, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := miner.MineNextBlock(chain, state, node.Mempool(), minerWallet.Address, 2_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Found {
		t.Fatal("failed to mine transaction block")
	}
	node.RefreshChainStatus()

	confirmed := request(t, server, http.MethodGet, "/v1/transactions/"+tx.ID, nil, "")
	if confirmed.Code != http.StatusOK {
		t.Fatalf("confirmed status returned %d: %s", confirmed.Code, confirmed.Body.String())
	}
	var confirmedStatus transactionStatusResponse
	if err := json.Unmarshal(confirmed.Body.Bytes(), &confirmedStatus); err != nil {
		t.Fatal(err)
	}
	if confirmedStatus.Status != "confirmed" || confirmedStatus.BlockHeight == nil || confirmedStatus.Confirmations < 1 {
		t.Fatalf("unexpected confirmed status: %+v", confirmedStatus)
	}
}

func TestTransactionStatusValidationAndMissing(t *testing.T) {
	server, _, _, _ := newTestServer(t)
	bad := request(t, server, http.MethodGet, "/v1/transactions/not-a-tx", nil, "")
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("invalid ID returned %d", bad.Code)
	}
	missingID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	missing := request(t, server, http.MethodGet, "/v1/transactions/"+missingID, nil, "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing ID returned %d", missing.Code)
	}
}
