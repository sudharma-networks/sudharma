package rpc

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// MiningStagingChallengeProvider supplies explicit non-consensus challenge
// parameters. The cache-node count is supplied on every challenge so this API
// never selects or freezes the production cache/DAG policy.
type MiningStagingChallengeProvider func() (headerPrefix []byte, height uint64, cacheNodes uint32, target []byte, err error)

type MiningStagingHTTPAPI struct {
	service  *MiningStagingService
	provider MiningStagingChallengeProvider
	gate     MiningRequestGate
}

// NewMiningStagingHTTPAPI creates a staging-only hardware interoperability
// surface. It deliberately does not expose live mining work or administrative
// RPC endpoints.
func NewMiningStagingHTTPAPI(service *MiningStagingService, provider MiningStagingChallengeProvider, gate MiningRequestGate) *MiningStagingHTTPAPI {
	return &MiningStagingHTTPAPI{service: service, provider: provider, gate: gate}
}

func (a *MiningStagingHTTPAPI) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")

		if a == nil || a.service == nil {
			writeError(w, http.StatusServiceUnavailable, "staging mining service unavailable")
			return
		}
		if a.gate != nil && !a.gate(r) {
			writeError(w, http.StatusTooManyRequests, "staging mining request rate limited")
			return
		}

		switch r.URL.Path {
		case "/v1/mining/staging/challenge":
			a.handleChallenge(w, r)
		case "/v1/mining/staging/submit":
			a.handleSubmit(w, r)
		default:
			writeError(w, http.StatusNotFound, "endpoint not found")
		}
	})
}

func (a *MiningStagingHTTPAPI) handleChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	if a.provider == nil {
		writeError(w, http.StatusServiceUnavailable, "staging challenge provider unavailable")
		return
	}

	headerPrefix, height, cacheNodes, target, err := a.provider()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "staging challenge unavailable")
		return
	}
	challenge, err := a.service.Issue(headerPrefix, height, cacheNodes, target)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, challenge)
}

func (a *MiningStagingHTTPAPI) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, miningHTTPMaxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var solution MiningStagingSolution
	if err := decoder.Decode(&solution); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body exceeds maximum size")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid staging mining solution JSON")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request must contain exactly one JSON object")
		return
	}

	result := a.service.Submit(solution)
	switch result.Status {
	case MiningSubmitAccepted:
		writeJSON(w, http.StatusOK, result)
	case MiningSubmitInvalid:
		writeJSON(w, http.StatusUnprocessableEntity, result)
	case MiningSubmitStale:
		writeJSON(w, http.StatusConflict, result)
	case MiningSubmitMutated:
		writeJSON(w, http.StatusBadRequest, result)
	default:
		writeError(w, http.StatusInternalServerError, "unknown staging mining submission status")
	}
}
