package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCUDASearchKernelContract(t *testing.T) {
	sourcePath := filepath.Join("..", "cuda", "gpupow_v1.cu")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read CUDA source: %v", err)
	}
	text := string(source)

	required := []string{
		"__global__ void khushi_search_kernel",
		"atomicMin",
		"stale_generation",
		"nonce_start",
		"nonce_count",
		"--benchmark",
		"Khushi Algorithm",
	}
	for _, token := range required {
		if !strings.Contains(text, token) {
			t.Fatalf("CUDA mining source missing required search contract token %q", token)
		}
	}

	if !strings.Contains(text, "CPU fallback prohibited") {
		t.Fatal("CUDA miner must preserve explicit no-CPU-fallback behavior")
	}
}
