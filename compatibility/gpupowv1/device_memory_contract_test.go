package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCUDAMinerDeviceEligibilityIsAllocationDriven(t *testing.T) {
	path := filepath.Join("..", "cuda", "khushi_miner.cu")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read CUDA miner source: %v", err)
	}
	text := string(data)

	required := []string{
		"required_vram_bytes",
		"available_vram_bytes",
		"cudaMemGetInfo",
		"--list-devices",
		"--device",
		"selected_device",
		"insufficient GPU memory",
	}
	for _, token := range required {
		if !strings.Contains(text, token) {
			t.Fatalf("CUDA miner missing model-neutral device contract token %q", token)
		}
	}

	lower := strings.ToLower(text)
	if strings.Contains(lower, "rtx2060-benchmark") {
		t.Fatal("benchmark input must not encode a specific GPU model")
	}
	if strings.Contains(lower, "totalglobalmem >= 4") || strings.Contains(lower, "4gb required") {
		t.Fatal("device eligibility must derive from actual allocation requirements, not a hard-coded 4 GB model rule")
	}
}
