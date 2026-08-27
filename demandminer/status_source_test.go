package demandminer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPStatusSourceDecodesNodeStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/status" {
			t.Fatalf("path = %s, want /v1/status", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"network":"sudharma","coin":"Sudharma","symbol":"SUDH","height":42,"issued_supply":123456,"mempool":3}`))
	}))
	defer server.Close()

	source := NewHTTPStatusSource(server.URL+"/v1/status", server.Client())
	got, err := source.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	want := (Status{Network: "sudharma", Coin: "Sudharma", Symbol: "SUDH", Height: 42, IssuedSupply: 123456, Mempool: 3})
	if got != want {
		t.Fatalf("Status = %+v, want %+v", got, want)
	}
}

func TestHTTPStatusSourceRejectsNonSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	source := NewHTTPStatusSource(server.URL, server.Client())
	_, err := source.Status(context.Background())
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected HTTP status error, got %v", err)
	}
}

func TestHTTPStatusSourceRejectsMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"network":`))
	}))
	defer server.Close()

	source := NewHTTPStatusSource(server.URL, server.Client())
	_, err := source.Status(context.Background())
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestHTTPStatusSourceHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	source := NewHTTPStatusSource("http://127.0.0.1:1/v1/status", http.DefaultClient)
	_, err := source.Status(ctx)
	if err == nil {
		t.Fatal("expected canceled request error")
	}
}
