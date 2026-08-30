package rpc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestMinerWakeEndpointAcceptsPost(t *testing.T) {
	server, _, _, _ := newTestServer(t)
	response := request(t, server, http.MethodPost, "/v1/miner/wake", nil, "")
	if response.Code != http.StatusAccepted {
		t.Fatalf("miner wake status: got %d", response.Code)
	}
}

func TestSubmitTransactionNotifiesDemandMinerWakeURL(t *testing.T) {
	var wakeCalls atomic.Int32
	wakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("wake method = %s", r.Method)
		}
		wakeCalls.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer wakeServer.Close()

	server, node, _, state := newTestServer(t)
	server.config.MinerWakeURL = wakeServer.URL

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
	body, err := json.Marshal(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := request(t, server, http.MethodPost, "/v1/transactions", body, "application/json")
	if response.Code != http.StatusAccepted {
		t.Fatalf("submit status: got %d body=%s", response.Code, response.Body.String())
	}
	if node.MempoolCount() != 1 {
		t.Fatalf("mempool count = %d", node.MempoolCount())
	}

	deadline := time.Now().Add(2 * time.Second)
	for wakeCalls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if wakeCalls.Load() != 1 {
		t.Fatalf("wake calls = %d", wakeCalls.Load())
	}
}
