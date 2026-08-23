package rpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/p2p"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func newTestServer(t *testing.T) (*Server, *p2p.Node, *blockchain.Chain, *blockchain.State) {
	t.Helper()
	chain := blockchain.NewChain()
	state := blockchain.NewState()
	node, err := p2p.NewNode("rpc-test-node", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := node.SetChain(chain); err != nil {
		t.Fatal(err)
	}
	if err := node.SetState(state); err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	server, err := NewServer(config, node, chain, state)
	if err != nil {
		t.Fatal(err)
	}
	return server, node, chain, state
}

func request(t *testing.T, server *Server, method, path string, body []byte, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, req)
	return recorder
}

func TestHealthAndSecurityHeaders(t *testing.T) {
	server, _, _, _ := newTestServer(t)
	response := request(t, server, http.MethodGet, "/health", nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("health status: got %d", response.Code)
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff header")
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing no-store cache policy")
	}
}

func TestStatusAndGenesisBlock(t *testing.T) {
	server, _, chain, _ := newTestServer(t)
	status := request(t, server, http.MethodGet, "/v1/status", nil, "")
	if status.Code != http.StatusOK {
		t.Fatalf("status endpoint returned %d", status.Code)
	}
	var decoded statusResponse
	if err := json.Unmarshal(status.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.NodeID != "rpc-test-node" || decoded.Height != chain.Height() || decoded.Symbol != params.CoinSymbol {
		t.Fatalf("unexpected status response: %+v", decoded)
	}

	block := request(t, server, http.MethodGet, "/v1/blocks/0", nil, "")
	if block.Code != http.StatusOK {
		t.Fatalf("genesis endpoint returned %d", block.Code)
	}
	missing := request(t, server, http.MethodGet, "/v1/blocks/999", nil, "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing block returned %d", missing.Code)
	}
}

func TestAccountEndpoint(t *testing.T) {
	server, _, _, state := newTestServer(t)
	if err := state.Credit("account-a", 12345); err != nil {
		t.Fatal(err)
	}
	if err := state.SetAccountNonce("account-a", 7); err != nil {
		t.Fatal(err)
	}
	response := request(t, server, http.MethodGet, "/v1/accounts/account-a", nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("account endpoint returned %d", response.Code)
	}
	var decoded accountResponse
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Balance != 12345 || decoded.ConfirmedNonce != 7 || decoded.NextNonce != 8 {
		t.Fatalf("unexpected account response: %+v", decoded)
	}
}

func TestSubmitTransaction(t *testing.T) {
	server, node, _, state := newTestServer(t)
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
		t.Fatalf("submit returned %d: %s", response.Code, response.Body.String())
	}
	if node.MempoolCount() != 1 {
		t.Fatalf("expected mempool count 1, got %d", node.MempoolCount())
	}
	if _, ok := node.MempoolTransaction(tx.ID); !ok {
		t.Fatal("submitted transaction not found in mempool")
	}

	duplicate := request(t, server, http.MethodPost, "/v1/transactions", body, "application/json")
	if duplicate.Code != http.StatusUnprocessableEntity {
		t.Fatalf("duplicate returned %d", duplicate.Code)
	}
}

func TestTransactionRequestHardening(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	wrongType := request(t, server, http.MethodPost, "/v1/transactions", []byte(`{}`), "text/plain")
	if wrongType.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("wrong content type returned %d", wrongType.Code)
	}

	unknownField := request(t, server, http.MethodPost, "/v1/transactions", []byte(`{"ID":"x","unexpected":true}`), "application/json")
	if unknownField.Code != http.StatusBadRequest {
		t.Fatalf("unknown field returned %d", unknownField.Code)
	}

	extraJSON := request(t, server, http.MethodPost, "/v1/transactions", []byte(`{} {}`), "application/json")
	if extraJSON.Code != http.StatusBadRequest {
		t.Fatalf("extra JSON returned %d", extraJSON.Code)
	}

	server.config.MaxBodyBytes = 32
	oversized := request(t, server, http.MethodPost, "/v1/transactions", []byte(strings.Repeat("x", 128)), "application/json")
	if oversized.Code != http.StatusBadRequest {
		t.Fatalf("oversized body returned %d", oversized.Code)
	}
}

func TestMethodAndMempoolLimitValidation(t *testing.T) {
	server, _, _, _ := newTestServer(t)
	method := request(t, server, http.MethodPost, "/v1/status", nil, "")
	if method.Code != http.StatusMethodNotAllowed || method.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("method handling failed: code=%d allow=%q", method.Code, method.Header().Get("Allow"))
	}
	badLimit := request(t, server, http.MethodGet, "/v1/mempool?limit=501", nil, "")
	if badLimit.Code != http.StatusBadRequest {
		t.Fatalf("bad mempool limit returned %d", badLimit.Code)
	}
}
