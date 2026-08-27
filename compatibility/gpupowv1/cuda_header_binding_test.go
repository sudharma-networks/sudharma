package gpupowv1

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCUDAHeaderBindingMatchesReferenceVector(t *testing.T) {
	compiler, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ is not available")
	}

	source := filepath.Join("..", "cuda", "gpupow_v1.cu")
	binary := filepath.Join(t.TempDir(), "sudharma-gpupow-cuda-header-binding")
	cmd := exec.Command(compiler, "-std=c++17", "-O2", "-x", "c++", source, "-o", binary)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile CUDA host-contract source: %v\n%s", err, out)
	}

	probe := exec.Command(
		binary,
		"--header-seed",
		"7375646861726d612d6770752d706f772d76312d7265666572656e63652d686561646572",
		"0123456789abcdef",
	)
	out, err := probe.CombinedOutput()
	if err != nil {
		t.Fatalf("CUDA header-binding probe failed: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	want := "header-digest=b0d0aff1d79715f94b3605db0f0c14b0fbde6aa4101f9dbcf07d0bbb44248118 work-seed=f91597d7f1afd0b0"
	if got != want {
		t.Fatalf("CUDA header-binding mismatch:\ngot:  %s\nwant: %s", got, want)
	}
}
