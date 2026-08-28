package pow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGPUV1CUDARegisterPermutationsMatchReference(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable CUDA compatibility compile test uses g++")
	}
	gpp, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ unavailable")
	}

	tempDir := t.TempDir()
	harness := filepath.Join(tempDir, "permutation_harness.cpp")
	binary := filepath.Join(tempDir, "sudharma-gpupow-permutation")
	header := filepath.Join("..", "compatibility", "cuda", "gpupow_v1_permutation.h")

	source := `#include <cstdint>
#include <cstdio>
#include "gpupow_v1_permutation.h"

int main(int argc, char** argv) {
    if (argc != 3) return 64;
    unsigned seed_lo = 0, seed_hi = 0;
    if (std::sscanf(argv[1], "%x", &seed_lo) != 1 || std::sscanf(argv[2], "%x", &seed_hi) != 1) return 65;
    const auto schedules = sudharma::gpupowv1::register_permutations(seed_lo, seed_hi);
    std::fputs("dst=", stdout);
    for (std::size_t i = 0; i < schedules.first.size(); ++i) {
        if (i) std::fputc(',', stdout);
        std::printf("%u", schedules.first[i]);
    }
    std::fputs(" src=", stdout);
    for (std::size_t i = 0; i < schedules.second.size(); ++i) {
        if (i) std::fputc(',', stdout);
        std::printf("%u", schedules.second[i]);
    }
    std::fputc('\n', stdout);
    return 0;
}
`
	if err := os.WriteFile(harness, []byte(source), 0o600); err != nil {
		t.Fatalf("write CUDA permutation harness: %v", err)
	}

	build := exec.Command(gpp, "-std=c++17", "-I"+filepath.Dir(header), harness, "-o", binary)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compile CUDA permutation compatibility harness: %v\n%s", err, output)
	}

	vectors := [][2]uint32{
		{0, 0},
		{0x01234567, 0x89abcdef},
		{0xffffffff, 0x00000001},
		{0xdeadbeef, 0xcafebabe},
	}
	for _, vector := range vectors {
		dst, src := gpuV1RegisterPermutations(vector[0], vector[1])
		wantDst := make([]string, 0, GPUV1NumRegs)
		wantSrc := make([]string, 0, GPUV1NumRegs)
		for i := range dst {
			wantDst = append(wantDst, fmt.Sprintf("%d", dst[i]))
			wantSrc = append(wantSrc, fmt.Sprintf("%d", src[i]))
		}
		want := "dst=" + strings.Join(wantDst, ",") + " src=" + strings.Join(wantSrc, ",")

		cmd := exec.Command(binary, fmt.Sprintf("%08x", vector[0]), fmt.Sprintf("%08x", vector[1]))
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CUDA permutation harness failed for %08x/%08x: %v\n%s", vector[0], vector[1], err, output)
		}
		if got := strings.TrimSpace(string(output)); got != want {
			t.Fatalf("CUDA permutation mismatch for %08x/%08x:\ngot  %s\nwant %s", vector[0], vector[1], got, want)
		}
	}
}
