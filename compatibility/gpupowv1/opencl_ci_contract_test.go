package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCLCICompilesKhushiMinerSources(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "gpu-pow-v1-opencl-ci.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read OpenCL workflow: %v", err)
	}
	text := string(data)
	for _, token := range []string{
		"khushi_pow.cl",
		"khushi_miner_opencl.cpp",
		"opencl-headers",
		"ocl-icd-opencl-dev",
		"clang -x cl",
		"g++",
		"-lOpenCL",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("OpenCL CI workflow missing compile-gate token %q", token)
		}
	}
}
