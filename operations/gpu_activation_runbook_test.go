package operations

import (
	"os"
	"strings"
	"testing"
)

func TestGPUActivationRunbookRetainsSafetyBoundary(t *testing.T) {
	data, err := os.ReadFile("../docs/testnet/GPU_POW_ACTIVATION_REHEARSAL.md")
	if err != nil {
		t.Fatalf("read activation rehearsal runbook: %v", err)
	}
	text := string(data)
	for _, required := range []string{
		"Does Not Authorize Public Activation",
		"720-block",
		"Both Nodes Must Be Stopped",
		"Evidence Manifest",
		"Abort Before the Boundary",
		"Snapshot Recovery At or After the Boundary",
		"GPUV1TestnetActivationHeight remains disabled",
		"GPUV1MainnetActivationHeight remains disabled",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("runbook missing %q", required)
		}
	}
}
