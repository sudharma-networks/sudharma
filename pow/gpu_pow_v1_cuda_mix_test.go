package pow

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGPUV1CUDACompatibilityRandomMathAndMergeMatchReference(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable CUDA compatibility compile test uses g++")
	}
	gpp, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ unavailable")
	}

	source := filepath.Join("..", "compatibility", "cuda", "gpupow_v1.cu")
	binary := filepath.Join(t.TempDir(), "sudharma-gpupow-cuda")
	build := exec.Command(gpp, "-std=c++17", "-x", "c++", source, "-o", binary)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compile CUDA compatibility source as portable C++: %v\n%s", err, output)
	}

	vectors := []struct {
		a, b, selector uint32
	}{
		{0x01234567, 0x89abcdef, 0x00000000},
		{0xffffffff, 0x00000001, 0x12345678},
		{0x80000000, 0x7fffffff, 0xdeadbeef},
		{0x13579bdf, 0x2468ace0, 0xffffffff},
	}

	for _, vector := range vectors {
		for _, mode := range []string{"math", "merge"} {
			cmd := exec.Command(binary, "--mix-op", mode,
				fmt.Sprintf("%08x", vector.a),
				fmt.Sprintf("%08x", vector.b),
				fmt.Sprintf("%08x", vector.selector))
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s command failed for a=%08x b=%08x selector=%08x: %v\n%s",
					mode, vector.a, vector.b, vector.selector, err, output)
			}

			var want uint32
			if mode == "math" {
				want = gpuV1RandomMath(vector.a, vector.b, vector.selector)
			} else {
				want = gpuV1RandomMerge(vector.a, vector.b, vector.selector)
			}
			got := strings.TrimSpace(string(output))
			wantText := fmt.Sprintf("mix-result=%08x", want)
			if got != wantText {
				t.Fatalf("%s mismatch for a=%08x b=%08x selector=%08x: got %q want %q",
					mode, vector.a, vector.b, vector.selector, got, wantText)
			}
		}
	}
}
