package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCLKhushiMinerContract(t *testing.T) {
	kernelPath := filepath.Join("..", "opencl", "khushi_pow.cl")
	kernel, err := os.ReadFile(kernelPath)
	if err != nil {
		t.Fatalf("read OpenCL Khushi kernel: %v", err)
	}
	text := string(kernel)
	for _, token := range []string{
		"SUDHARMA_GPU_POW_V1_REFERENCE_HEADER",
		"SUDHARMA_GPU_POW_V1_FINAL",
		"KHUSHI_DAG_ROUNDS 64",
		"KHUSHI_DATASET_PARENTS 512",
		"nonce_start",
		"nonce_count",
		"stale_generation",
		"expected_generation",
		"khushi_search",
		"atomic_cmpxchg",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("OpenCL kernel missing consensus/search contract token %q", token)
		}
	}

	hostPath := filepath.Join("..", "opencl", "khushi_miner_opencl.cpp")
	host, err := os.ReadFile(hostPath)
	if err != nil {
		t.Fatalf("read OpenCL host miner: %v", err)
	}
	hostText := string(host)
	for _, token := range []string{
		"Khushi Algorithm",
		"--list-devices",
		"--device",
		"--vector-self-test",
		"--benchmark",
		"CL_DEVICE_GLOBAL_MEM_SIZE",
		"required_vram_bytes",
		"CPU fallback prohibited",
	} {
		if !strings.Contains(hostText, token) {
			t.Fatalf("OpenCL host miner missing multi-vendor contract token %q", token)
		}
	}
}
