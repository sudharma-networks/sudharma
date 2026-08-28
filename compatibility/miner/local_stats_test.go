package miner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestClientCountsLocallyStaleAndVerifierRejectedResults(t *testing.T) {
	var workCalls atomic.Int32
	var submits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/mining/work", func(w http.ResponseWriter, r *http.Request) {
		call := workCalls.Add(1)
		workID := "local-stats-a"
		if call > 1 {
			workID = "local-stats-b"
		}
		_ = json.NewEncoder(w).Encode(Work{
			WorkID: workID, Algorithm: "sudharma-gpupow-v1", Version: 2,
			Height: 101, Difficulty: 2, Target: "000f", HeaderPrefix: "7788", RewardAddress: "SUDH-local-stats",
		})
	})
	mux.HandleFunc("/v1/mining/submit", func(w http.ResponseWriter, r *http.Request) {
		submits.Add(1)
		_ = json.NewEncoder(w).Encode(SubmitResult{Status: "accepted"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	first, generation1, err := client.Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, generation2, err := client.Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.SubmitVerified(context.Background(), first, generation1, 1, func(Work, uint64) bool { return true }); err == nil {
		t.Fatal("stale GPU result must be rejected before submission")
	}
	if _, err := client.SubmitVerified(context.Background(), second, generation2, 2, func(Work, uint64) bool { return false }); err == nil {
		t.Fatal("host-verifier rejected result must not be submitted")
	}
	if submits.Load() != 0 {
		t.Fatalf("local stale/rejected results reached submit endpoint: %d", submits.Load())
	}

	stats := client.Stats()
	if stats.Stale != 1 || stats.Rejected != 1 || stats.Accepted != 0 {
		t.Fatalf("local mining result counters not recorded: %+v", stats)
	}
}
