package rpc

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/sudharma-networks/sudharma/blockchain"
)

const miningHTTPMaxBodyBytes = int64(64 << 10)

type MiningBlockProvider func() (*blockchain.Block, error)
type MiningRequestGate func(*http.Request) bool

type MiningTelemetry struct {
	WorkIssued     uint64 `json:"work_issued"`
	SubmitAccepted uint64 `json:"submit_accepted"`
	SubmitInvalid  uint64 `json:"submit_invalid"`
	SubmitStale    uint64 `json:"submit_stale"`
	SubmitMutated  uint64 `json:"submit_mutated"`
	RateLimited    uint64 `json:"rate_limited"`
}

type MiningHTTPAPI struct {
	service  *MiningWorkService
	provider MiningBlockProvider
	gate     MiningRequestGate

	workIssued     atomic.Uint64
	submitAccepted atomic.Uint64
	submitInvalid  atomic.Uint64
	submitStale    atomic.Uint64
	submitMutated  atomic.Uint64
	rateLimited    atomic.Uint64
}

// NewMiningHTTPAPI creates a deliberately narrow external-mining surface. Its
// handler exposes only get-work and submit-work; node administration remains on
// the normal RPC server and is not reachable through this handler.
func NewMiningHTTPAPI(service *MiningWorkService, provider MiningBlockProvider, gate MiningRequestGate) *MiningHTTPAPI {
	return &MiningHTTPAPI{service: service, provider: provider, gate: gate}
}

func (a *MiningHTTPAPI) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")

		if a == nil || a.service == nil {
			writeError(w, http.StatusServiceUnavailable, "mining service unavailable")
			return
		}
		if a.gate != nil && !a.gate(r) {
			a.rateLimited.Add(1)
			writeError(w, http.StatusTooManyRequests, "mining request rate limited")
			return
		}

		switch r.URL.Path {
		case "/v1/mining/work":
			a.handleGetWork(w, r)
		case "/v1/mining/submit":
			a.handleSubmitWork(w, r)
		default:
			writeError(w, http.StatusNotFound, "endpoint not found")
		}
	})
}

func (a *MiningHTTPAPI) handleGetWork(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	if a.provider == nil {
		writeError(w, http.StatusServiceUnavailable, "mining work provider unavailable")
		return
	}
	block, err := a.provider()
	if err != nil || block == nil {
		writeError(w, http.StatusServiceUnavailable, "mining work unavailable")
		return
	}
	work, err := a.service.Issue(block)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	a.workIssued.Add(1)
	writeJSON(w, http.StatusOK, work)
}

func (a *MiningHTTPAPI) handleSubmitWork(w http.ResponseWriter, r *http.Request) {
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
	var solution MiningSolution
	if err := decoder.Decode(&solution); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body exceeds maximum size")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid mining solution JSON")
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
		a.submitAccepted.Add(1)
		writeJSON(w, http.StatusOK, result)
	case MiningSubmitInvalid:
		a.submitInvalid.Add(1)
		writeJSON(w, http.StatusUnprocessableEntity, result)
	case MiningSubmitStale:
		a.submitStale.Add(1)
		writeJSON(w, http.StatusConflict, result)
	case MiningSubmitMutated:
		a.submitMutated.Add(1)
		writeJSON(w, http.StatusBadRequest, result)
	default:
		writeError(w, http.StatusInternalServerError, "unknown mining submission status")
	}
}

func (a *MiningHTTPAPI) Telemetry() MiningTelemetry {
	if a == nil {
		return MiningTelemetry{}
	}
	return MiningTelemetry{
		WorkIssued:     a.workIssued.Load(),
		SubmitAccepted: a.submitAccepted.Load(),
		SubmitInvalid:  a.submitInvalid.Load(),
		SubmitStale:    a.submitStale.Load(),
		SubmitMutated:  a.submitMutated.Load(),
		RateLimited:    a.rateLimited.Load(),
	}
}
