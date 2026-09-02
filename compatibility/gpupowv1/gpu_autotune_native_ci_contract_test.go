package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestGPUAutotuneNativeWorkflowsCoverSharedPolicy(t *testing.T) {
	for _, path := range []string{
		"../../.github/workflows/khushi-windows-cuda.yml",
		"../../.github/workflows/khushi-windows-opencl.yml",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, want := range []string{
			"feature/gpu-autotune-profiles",
			"compatibility/gpu/**",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s must cover the shared GPU autotune policy; missing %q", path, want)
			}
		}
	}
}

func TestCUDANativeArtifactIncludesBlackwellTarget(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/khushi-windows-cuda.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"compute_120,code=sm_120",
		"compute_120,code=compute_120",
		"sm_120",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("CUDA Windows artifact must retain native/PTX Blackwell support; missing %q", want)
		}
	}
}
