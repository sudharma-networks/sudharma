package gpupowv1

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCUDAProductionDatasetChunkMapping(t *testing.T) {
	compiler, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ is not available")
	}

	temp := t.TempDir()
	source := filepath.Join(temp, "chunks.cpp")
	binary := filepath.Join(temp, "chunks")
	program := `
#include <cstdint>
#include <cstdio>
#include "gpupow_v1_chunks.cuh"

int main() {
    const std::uint64_t indices[] = {0ull, 4194303ull, 4194304ull, 33554431ull};
    for (const auto index : indices) {
        sudharma::gpupowv1::DatasetLocation location{};
        if (!sudharma::gpupowv1::dataset_item_location(index, &location)) return 2;
        std::printf("%llu:%u:%llu\n",
            static_cast<unsigned long long>(index), location.chunk,
            static_cast<unsigned long long>(location.offset));
    }
    sudharma::gpupowv1::DatasetLocation invalid{};
    return sudharma::gpupowv1::dataset_item_location(33554432ull, &invalid) ? 3 : 0;
}
`
	if err := os.WriteFile(source, []byte(program), 0o600); err != nil {
		t.Fatal(err)
	}
	include := filepath.Join("..", "cuda")
	cmd := exec.Command(compiler, "-std=c++17", "-O2", "-I", include, source, "-o", binary)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile CUDA chunk contract: %v\n%s", err, out)
	}
	out, err := exec.Command(binary).CombinedOutput()
	if err != nil {
		t.Fatalf("run CUDA chunk contract: %v\n%s", err, out)
	}
	want := "0:0:0\n4194303:0:268435392\n4194304:1:0\n33554431:7:268435392"
	if got := strings.TrimSpace(string(out)); got != want {
		t.Fatalf("CUDA chunk mapping mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
