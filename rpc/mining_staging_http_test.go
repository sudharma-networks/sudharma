package rpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiningStagingHTTPAPIOnlyExposesChallengeAndSubmit(t *testing.T) {
	service := NewMiningStagingService(func(challenge MiningStagingChallenge, nonce uint64) bool {
		return nonce == 7
	})
	provider := func() ([]byte, uint64, uint32, []byte, error) {
		return []byte("hardware-gate"), 0, 8, bytes.Repeat([]byte{0xff}, 32), nil
	}
	api := NewMiningStagingHTTPAPI(service, provider, nil)

	workRequest := httptest.NewRequest(http.MethodGet, "/v1/mining/work", nil)
	workRecorder := httptest.NewRecorder()
	api.Handler().ServeHTTP(workRecorder, workRequest)
	if workRecorder.Code != http.StatusNotFound {
		t.Fatalf("live mining endpoint status: got %d want %d", workRecorder.Code, http.StatusNotFound)
	}

	challengeRequest := httptest.NewRequest(http.MethodGet, "/v1/mining/staging/challenge", nil)
	challengeRecorder := httptest.NewRecorder()
	api.Handler().ServeHTTP(challengeRecorder, challengeRequest)
	if challengeRecorder.Code != http.StatusOK {
		t.Fatalf("challenge status: got %d want %d", challengeRecorder.Code, http.StatusOK)
	}
	var challenge MiningStagingChallenge
	if err := json.NewDecoder(challengeRecorder.Body).Decode(&challenge); err != nil {
		t.Fatal(err)
	}
	if !challenge.Staging || challenge.CacheNodes != 8 {
		t.Fatal("HTTP challenge must remain explicit non-consensus staging work")
	}

	body, err := json.Marshal(MiningStagingSolution{Challenge: challenge, Nonce: 7})
	if err != nil {
		t.Fatal(err)
	}
	submitRequest := httptest.NewRequest(http.MethodPost, "/v1/mining/staging/submit", bytes.NewReader(body))
	submitRequest.Header.Set("Content-Type", "application/json")
	submitRecorder := httptest.NewRecorder()
	api.Handler().ServeHTTP(submitRecorder, submitRequest)
	if submitRecorder.Code != http.StatusOK {
		t.Fatalf("submit status: got %d want %d body=%s", submitRecorder.Code, http.StatusOK, submitRecorder.Body.String())
	}
	var result MiningSubmitResult
	if err := json.NewDecoder(submitRecorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Status != MiningSubmitAccepted {
		t.Fatalf("submit result: got %q want %q", result.Status, MiningSubmitAccepted)
	}
}

func TestMiningStagingHTTPAPIStrictlyRejectsUnknownSubmissionFields(t *testing.T) {
	service := NewMiningStagingService(func(MiningStagingChallenge, uint64) bool { return false })
	api := NewMiningStagingHTTPAPI(service, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/v1/mining/staging/submit", bytes.NewBufferString(`{"challenge":{},"nonce":0,"unexpected":true}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	api.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}
}
