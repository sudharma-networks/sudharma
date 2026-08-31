package gpuminer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/params"
)

func TestCommandBackendUsesGPUHasherAndRejectsCPUOutput(t *testing.T) {
	backend := CommandBackend{
		Path: "khushi-miner-nvidia",
		Run: func(_ context.Context, path string, args []string) ([]byte, error) {
			if path != "khushi-miner-nvidia" {
				t.Fatalf("path = %q", path)
			}
			joined := strings.Join(args, " ")
			if !strings.Contains(joined, "--staging-search") || !strings.Contains(joined, "--header-prefix-hex aa") {
				t.Fatalf("args = %v", args)
			}
			if strings.Contains(joined, "cpu") {
				t.Fatal("must not pass a CPU mining flag")
			}
			return []byte("Khushi GPU search\nstaging-solution-nonce=42\n"), nil
		},
	}
	nonce, err := backend.Search(context.Background(), Work{
		Algorithm:    params.ProductionMiningAlgorithm,
		HeaderPrefix: "aa",
		Target:       "0f",
		Height:       12,
	})
	if err != nil {
		t.Fatal(err)
	}
	if nonce != 42 {
		t.Fatalf("nonce = %d", nonce)
	}

	if _, err := ParseHasherNonce([]byte("hashrate_hps=1\n")); err == nil {
		t.Fatal("CPU-style output must not yield a nonce")
	}
}

func TestGetWorkFallsBackToGETWhenPOSTIsMissing(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			http.Error(w, `{"error":"route not found"}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(Work{
			WorkID:        "work-get",
			Algorithm:     params.ProductionMiningAlgorithm,
			Version:       2,
			Height:        9,
			Target:        "0f",
			HeaderPrefix:  "aa",
			RewardAddress: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	work, err := client.GetWork(context.Background(), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	if work.Height != 9 {
		t.Fatalf("height = %d", work.Height)
	}
	if strings.Join(methods, ",") != "POST,GET" {
		t.Fatalf("methods = %v", methods)
	}
}

func TestHasherArgsStayOnKhushiGPUSearch(t *testing.T) {
	args := HasherArgs(Work{HeaderPrefix: "aa", Target: "bb", Height: 4, CacheNodes: 8}, 1)
	got := strings.Join(args, " ")
	if !strings.Contains(got, "--device 1") || !strings.Contains(got, "--staging-search") {
		t.Fatalf("args = %v", args)
	}
	if bytes.Contains([]byte(got), []byte("--mine")) {
		t.Fatal("must not ungate --mine")
	}
}
