package rpc

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestMiningEndpointsAreGPUOnlyAndDoNotIssueCPUWork(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	work := request(t, server, http.MethodPost, "/v1/mining/work", []byte(`{"address":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`), "application/json")
	if work.Code != http.StatusServiceUnavailable {
		t.Fatalf("work status = %d", work.Code)
	}
	var payload errorResponse
	if err := json.Unmarshal(work.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload.Error, "GPU-only") {
		t.Fatalf("work error = %q", payload.Error)
	}
	if strings.Contains(payload.Error, "sudharma-cpu-v1") {
		t.Fatal("must not advertise a CPU mining algorithm")
	}

	getWork := request(t, server, http.MethodGet, "/v1/mining/work?address=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", nil, "")
	if getWork.Code != http.StatusServiceUnavailable {
		t.Fatalf("get work status = %d", getWork.Code)
	}

	submit := request(t, server, http.MethodPost, "/v1/mining/submit", []byte(`{"algorithm":"sudharma-cpu-v1"}`), "application/json")
	if submit.Code != http.StatusServiceUnavailable {
		t.Fatalf("submit status = %d", submit.Code)
	}
	if !strings.Contains(submit.Body.String(), params.GPUOnlyMiningMessage) {
		t.Fatalf("submit body = %q", submit.Body.String())
	}
}

func TestMiningEndpointsRejectWrongMethods(t *testing.T) {
	server, _, _, _ := newTestServer(t)
	put := request(t, server, http.MethodPut, "/v1/mining/work", nil, "")
	if put.Code != http.StatusMethodNotAllowed {
		t.Fatalf("put work status = %d", put.Code)
	}
	getSubmit := request(t, server, http.MethodGet, "/v1/mining/submit", nil, "")
	if getSubmit.Code != http.StatusMethodNotAllowed {
		t.Fatalf("get submit status = %d", getSubmit.Code)
	}
}
