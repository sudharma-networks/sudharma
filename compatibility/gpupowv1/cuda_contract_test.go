package gpupowv1

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCUDAMinerHostSelfTestAndNoCPUFallback(t *testing.T) {
	compiler, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ is not available")
	}

	source := filepath.Join("..", "cuda", "gpupow_v1.cu")
	binary := filepath.Join(t.TempDir(), "sudharma-gpupow-cuda-contract")
	cmd := exec.Command(compiler, "-std=c++17", "-O2", "-x", "c++", source, "-o", binary)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile CUDA host-contract source: %v\n%s", err, out)
	}

	selfTest := exec.Command(binary, "--self-test")
	out, err := selfTest.CombinedOutput()
	if err != nil {
		t.Fatalf("CUDA host self-test failed: %v\n%s", err, out)
	}
	got := strings.TrimSpace(string(out))
	want := "kiss99=5f502f5e,5065034f,0b7649f5,6759296d\nself-test=ok"
	if got != want {
		t.Fatalf("CUDA primitive self-test mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}

	mine := exec.Command(binary, "--mine")
	out, err = mine.CombinedOutput()
	if err == nil {
		t.Fatal("non-CUDA build must refuse mining instead of silently falling back to CPU")
	}
	if !strings.Contains(string(out), "CUDA backend required; CPU fallback prohibited") {
		t.Fatalf("unexpected non-CUDA mining refusal: %s", out)
	}
}
