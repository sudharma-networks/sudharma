package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCUDATelemetryContract(t *testing.T) {
	path := filepath.Join("..", "cuda", "khushi_miner.cu")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read CUDA miner: %v", err)
	}
	text := string(data)
	required := []string{
		"--telemetry",
		"nvidia-smi",
		"temperature.gpu",
		"power.draw",
		"utilization.gpu",
		"memory.used",
	}
	for _, token := range required {
		if !strings.Contains(text, token) {
			t.Fatalf("CUDA miner missing telemetry token %q", token)
		}
	}
}
