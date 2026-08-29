package rpc

import (
	"encoding/json"
	"net/http"
	"testing"
)

type gpuActivationHTTPEnvelope struct {
	GPUV1 struct {
		Phase            string  `json:"phase"`
		ActivationHeight *uint64 `json:"activation_height,omitempty"`
		NextBlockVersion uint32  `json:"next_block_version"`
	} `json:"gpu_v1"`
}

func TestStatusAndReadyExposeSafeDisabledGPUActivationState(t *testing.T) {
	server, _, _, _ := newTestServer(t)
	for _, path := range []string{"/v1/status", "/ready"} {
		t.Run(path, func(t *testing.T) {
			response := request(t, server, http.MethodGet, path, nil, "")
			if response.Code != http.StatusOK {
				t.Fatalf("%s returned %d: %s", path, response.Code, response.Body.String())
			}
			var decoded gpuActivationHTTPEnvelope
			if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.GPUV1.Phase != "disabled" {
				t.Fatalf("%s gpu_v1 phase = %q, want disabled; body=%s", path, decoded.GPUV1.Phase, response.Body.String())
			}
			if decoded.GPUV1.ActivationHeight != nil {
				t.Fatalf("%s exposed disabled activation height %d", path, *decoded.GPUV1.ActivationHeight)
			}
			if decoded.GPUV1.NextBlockVersion != 1 {
				t.Fatalf("%s next block version = %d, want 1", path, decoded.GPUV1.NextBlockVersion)
			}
		})
	}
}
