package rpc

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestStatusIncludesCanonicalNetworkID(t *testing.T) {
	server, _, chain, _ := newTestServer(t)
	response := request(t, server, http.MethodGet, "/v1/status", nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("status endpoint returned %d", response.Code)
	}

	var decoded map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if got := decoded["network"]; got != "sudharma" {
		t.Fatalf("legacy network label changed: got %v", got)
	}
	if got := decoded["network_id"]; got != string(chain.Network()) {
		t.Fatalf("network_id = %v, want %q", got, chain.Network())
	}
}
