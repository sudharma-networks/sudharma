package miner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestClientPollGenerationAndVerifiedSubmit(t *testing.T) {
	var workCalls atomic.Int32
	var submitCalls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/mining/work", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("work method = %s", r.Method)
		}
		call := workCalls.Add(1)
		workID := "work-a"
		if call > 1 {
			workID = "work-b"
		}
		_ = json.NewEncoder(w).Encode(Work{
			WorkID: workID, Algorithm: "sudharma-gpupow-v1", Version: 2,
			Height: 42, Difficulty: 1, Target: "01", HeaderPrefix: "00", RewardAddress: "SUDH-test",
		})
	})
	mux.HandleFunc("/v1/mining/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("submit method = %s", r.Method)
		}
		submitCalls.Add(1)
		var got Solution
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode submit: %v", err)
		}
		if got.WorkID != "work-b" || got.Nonce != 99 {
			t.Fatalf("unexpected submitted solution: %+v", got)
		}
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
	if first.WorkID == second.WorkID || generation2 <= generation1 {
		t.Fatalf("generation did not roll over: %d -> %d", generation1, generation2)
	}
	if !client.IsCurrent(second.WorkID, generation2) || client.IsCurrent(first.WorkID, generation1) {
		t.Fatal("stale generation binding is incorrect")
	}

	verified := false
	result, err := client.SubmitVerified(context.Background(), second, generation2, 99, func(work Work, nonce uint64) bool {
		verified = true
		return work.WorkID == "work-b" && nonce == 99
	})
	if err != nil {
		t.Fatal(err)
	}
	if !verified || result.Status != "accepted" || submitCalls.Load() != 1 {
		t.Fatalf("verified submit failed: verified=%v result=%+v submits=%d", verified, result, submitCalls.Load())
	}
}

func TestClientRejectsReusedWorkIDWithMutatedTemplate(t *testing.T) {
	var calls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/mining/work", func(w http.ResponseWriter, r *http.Request) {
		work := Work{
			WorkID: "stable-id", Algorithm: "sudharma-gpupow-v1", Version: 2,
			Height: 77, Difficulty: 3, Target: "000f", HeaderPrefix: "aabb", RewardAddress: "SUDH-miner",
		}
		if calls.Add(1) > 1 {
			work.HeaderPrefix = "ccdd"
		}
		_ = json.NewEncoder(w).Encode(work)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	first, generation, err := client.Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.Poll(context.Background()); err == nil {
		t.Fatal("reused work_id with mutated immutable template must be rejected")
	}
	if !client.IsCurrent(first.WorkID, generation) {
		t.Fatal("rejecting mutated work must preserve the previously accepted work binding")
	}
}

func TestClientNeverSubmitsWithoutVerifierApproval(t *testing.T) {
	var submits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/mining/work", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Work{WorkID: "w", Algorithm: "sudharma-gpupow-v1", Version: 2, Height: 1, Difficulty: 1, Target: "01", HeaderPrefix: "00", RewardAddress: "x"})
	})
	mux.HandleFunc("/v1/mining/submit", func(w http.ResponseWriter, r *http.Request) {
		submits.Add(1)
		_ = json.NewEncoder(w).Encode(SubmitResult{Status: "accepted"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client, _ := NewClient(srv.URL, srv.Client())
	work, generation, _ := client.Poll(context.Background())
	if _, err := client.SubmitVerified(context.Background(), work, generation, 7, func(Work, uint64) bool { return false }); err == nil {
		t.Fatal("candidate rejected by host verifier must not be submitted")
	}
	if submits.Load() != 0 {
		t.Fatalf("unverified solution reached submit endpoint: %d", submits.Load())
	}
}

func TestClientRejectsAdministrativeBasePathAndOversizedWork(t *testing.T) {
	if _, err := NewClient("https://example.test/admin", http.DefaultClient); err == nil {
		t.Fatal("mining client base URL must not include an administrative path")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(make([]byte, maxResponseBytes+1))
	}))
	defer srv.Close()
	client, _ := NewClient(srv.URL, srv.Client())
	if _, _, err := client.Poll(context.Background()); err == nil {
		t.Fatal("oversized mining work response must be rejected")
	}
}
