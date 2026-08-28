package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCLSearchUsesCL12WinnerFlag(t *testing.T) {
	path := filepath.Join("..", "opencl", "khushi_pow.cl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Khushi OpenCL kernel: %v", err)
	}
	text := string(data)

	for _, token := range []string{
		"volatile __global uint* found_flag",
		"atomic_cmpxchg(found_flag, 0u, 1u)",
		"*found_nonce = nonce",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("OpenCL CL1.2 winner contract missing %q", token)
		}
	}

	if strings.Contains(text, "cl_khr_int64_base_atomics") {
		t.Fatal("OpenCL search must not require optional 64-bit atomic extension")
	}
	if strings.Contains(text, "atomic_cmpxchg(slot,observed,nonce)") {
		t.Fatal("OpenCL search must not use ambiguous 64-bit atomic_cmpxchg")
	}
}
