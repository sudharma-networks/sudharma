package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sudharma-networks/sudharma/pow"
)

func testDigestAtOrBelowTarget(digest [32]byte, target []byte) bool {
	if len(target) != len(digest) {
		return false
	}
	for i := range digest {
		if digest[i] < target[i] {
			return true
		}
		if digest[i] > target[i] {
			return false
		}
	}
	return true
}

func solveStagingChallenge(t *testing.T, challenge stagingChallenge) uint64 {
	t.Helper()
	header, err := hex.DecodeString(challenge.HeaderPrefix)
	if err != nil {
		t.Fatalf("decode header: %v", err)
	}
	target, err := hex.DecodeString(challenge.Target)
	if err != nil {
		t.Fatalf("decode target: %v", err)
	}
	cache := pow.GPUV1BuildCache(pow.GPUV1EpochSeed(pow.GPUV1EpochForHeight(challenge.Height)), challenge.CacheNodes)
	for nonce := uint64(0); nonce < 1_000_000; nonce++ {
		digest := pow.GPUV1ReferenceDigest(header, nonce, challenge.Height, cache)
		if testDigestAtOrBelowTarget(digest, target) {
			return nonce
		}
	}
	t.Fatal("unable to solve compact staging target")
	return 0
}

func getStagingChallenge(t *testing.T, handler http.Handler) stagingChallenge {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/mining/staging/challenge", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("challenge status=%d body=%s", rec.Code, rec.Body.String())
	}
	var challenge stagingChallenge
	if err := json.NewDecoder(rec.Body).Decode(&challenge); err != nil {
		t.Fatalf("decode challenge: %v", err)
	}
	if !challenge.Staging || challenge.Algorithm != pow.GPUV1AlgorithmID {
		t.Fatalf("unexpected challenge: %+v", challenge)
	}
	if challenge.Height != 0 || challenge.CacheNodes != 8 {
		t.Fatalf("unexpected compact staging parameters: %+v", challenge)
	}
	return challenge
}

func submitStagingSolution(t *testing.T, handler http.Handler, challenge stagingChallenge, nonce uint64) stagingResult {
	t.Helper()
	body, err := json.Marshal(stagingSubmission{Challenge: challenge, Nonce: nonce})
	if err != nil {
		t.Fatalf("marshal submission: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/mining/staging/submit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result stagingResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	return result
}

func TestStagingAPIAcceptsCanonicalSolutionAndRejectsReplay(t *testing.T) {
	api := newStagingAPI()
	handler := api.handler()
	challenge := getStagingChallenge(t, handler)
	nonce := solveStagingChallenge(t, challenge)

	if got := submitStagingSolution(t, handler, challenge, nonce).Status; got != "accepted" {
		t.Fatalf("valid staging solution status=%q", got)
	}
	if got := submitStagingSolution(t, handler, challenge, nonce).Status; got != "rejected" {
		t.Fatalf("replayed staging solution status=%q, want rejected", got)
	}
}

func TestStagingAPIRejectsChallengeMutation(t *testing.T) {
	api := newStagingAPI()
	handler := api.handler()
	challenge := getStagingChallenge(t, handler)
	nonce := solveStagingChallenge(t, challenge)
	challenge.HeaderPrefix = "00" + challenge.HeaderPrefix
	if got := submitStagingSolution(t, handler, challenge, nonce).Status; got != "rejected" {
		t.Fatalf("mutated challenge status=%q, want rejected", got)
	}
}

func TestValidateListenAddressRequiresLoopback(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:28646", "localhost:28646", "127.0.0.1:0"} {
		if err := validateListenAddress(addr); err != nil {
			t.Errorf("loopback address %q rejected: %v", addr, err)
		}
	}
	for _, addr := range []string{"0.0.0.0:28646", ":28646", "192.0.2.10:28646"} {
		if err := validateListenAddress(addr); err == nil {
			t.Errorf("non-loopback address %q accepted", addr)
		}
	}
}
