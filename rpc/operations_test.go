package rpc

import (
	"net/http"
	"strings"
	"testing"
)

func TestReadinessEndpoint(t *testing.T) {
	server, _, chain, _ := newTestServer(t)
	response := request(t, server, http.MethodGet, "/ready", nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("ready status: got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), chain.Tip().Hash()) {
		t.Fatalf("ready response does not contain current tip: %s", response.Body.String())
	}
}

func TestMetricsEndpoint(t *testing.T) {
	server, _, _, _ := newTestServer(t)
	response := request(t, server, http.MethodGet, "/metrics", nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("metrics status: got %d", response.Code)
	}
	body := response.Body.String()
	for _, metric := range []string{"sudharma_chain_height", "sudharma_peers", "sudharma_mempool_transactions", "sudharma_issued_supply"} {
		if !strings.Contains(body, metric) {
			t.Fatalf("missing metric %s in %s", metric, body)
		}
	}
	if !strings.HasPrefix(response.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("unexpected metrics content type: %s", response.Header().Get("Content-Type"))
	}
}

func TestMetricsCanBeDisabled(t *testing.T) {
	server, node, chain, state := newTestServer(t)
	cfg := DefaultConfig()
	cfg.EnableMetrics = false
	disabled, err := NewServer(cfg, node, chain, state)
	if err != nil { t.Fatal(err) }
	response := request(t, disabled, http.MethodGet, "/metrics", nil, "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("disabled metrics returned %d", response.Code)
	}
	_ = server
}
