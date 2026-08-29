package rpc

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestExplorerBackportStatusRouteIsReadOnlyAndCanonical(t *testing.T) {
	server, node, chain, state := newTestServer(t)
	if err := state.MintSupply(123); err != nil {
		t.Fatal(err)
	}

	response := request(t, server, http.MethodGet, "/v1/explorer/status", nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("explorer status returned %d: %s", response.Code, response.Body.String())
	}

	var got struct {
		Network      string `json:"network"`
		Coin         string `json:"coin"`
		Symbol       string `json:"symbol"`
		Height       uint64 `json:"height"`
		TipHash      string `json:"tip_hash"`
		TotalWork    string `json:"total_work"`
		Peers        int    `json:"peers"`
		Mempool      int    `json:"mempool"`
		IssuedSupply uint64 `json:"issued_supply"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Network != "sudharma" || got.Coin != params.CoinName || got.Symbol != params.CoinSymbol {
		t.Fatalf("unexpected explorer identity: %+v", got)
	}
	if got.Height != chain.Height() || got.TipHash != chain.Tip().Hash() || got.TotalWork != chain.TotalWork().String() {
		t.Fatalf("unexpected explorer chain status: %+v", got)
	}
	if got.Peers != node.PeerCount() || got.Mempool != node.MempoolCount() || got.IssuedSupply != 123 {
		t.Fatalf("unexpected explorer counters: %+v", got)
	}

	post := request(t, server, http.MethodPost, "/v1/explorer/status", nil, "")
	if post.Code != http.StatusMethodNotAllowed || post.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("explorer POST handling = %d allow=%q", post.Code, post.Header().Get("Allow"))
	}
}
