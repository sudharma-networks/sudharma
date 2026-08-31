package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/gpuminer"
	"github.com/sudharma-networks/sudharma/params"
)

func TestRunRejectsCPUBackendAndMainnet(t *testing.T) {
	err := run([]string{
		"-address", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"-backend", "cpu",
		"-probe",
	}, strings.NewReader(""), ioDiscard(), ioDiscard())
	if err == nil || !strings.Contains(err.Error(), "GPU-only") {
		t.Fatalf("cpu backend error = %v", err)
	}

	err = run([]string{
		"-address", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"-network", "mainnet",
		"-probe",
	}, strings.NewReader(""), ioDiscard(), ioDiscard())
	if err == nil || !strings.Contains(err.Error(), "mainnet") {
		t.Fatalf("mainnet error = %v", err)
	}
}

func TestRunProbeConnectsWithoutCPUFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": params.GPUOnlyMiningMessage})
	}))
	t.Cleanup(server.Close)

	out := &strings.Builder{}
	err := run([]string{
		"-address", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"-rpc", server.URL,
		"-probe",
	}, strings.NewReader(""), out, ioDiscard())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "GPU-only") && !strings.Contains(out.String(), "GPU-PoW work is not being issued") {
		t.Fatalf("output = %q", out.String())
	}
	if strings.Contains(strings.ToLower(out.String()), "cpu mining started") {
		t.Fatal("must not start CPU mining")
	}
}

func TestRunPromptsForAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(gpuminer.Work{
			Algorithm:    params.ProductionMiningAlgorithm,
			Version:      2,
			Height:       3,
			Target:       "0f",
			HeaderPrefix: "aa",
		})
	}))
	t.Cleanup(server.Close)

	out := &strings.Builder{}
	err := run([]string{"-rpc", server.URL, "-probe"}, strings.NewReader("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n"), out, ioDiscard())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestDetectGPUHasher(t *testing.T) {
	dir := t.TempDir()
	if _, err := gpuminer.DetectGPUHasher(dir); err == nil {
		t.Fatal("expected missing hasher")
	}
	path := filepath.Join(dir, "khushi-miner-nvidia")
	if err := os.WriteFile(path, []byte("gpu"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := gpuminer.DetectGPUHasher(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("hasher = %q", got)
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

func ioDiscard() *discard { return &discard{} }
