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
	if got := strings.TrimSpace(strings.ReplaceAll(string(out), "\r\n", "\n")); got != want {
		t.Fatalf("CUDA chunk mapping mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestCUDAProductionDatasetChunkAllocationCleanup(t *testing.T) {
	compiler, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ is not available")
	}

	temp := t.TempDir()
	source := filepath.Join(temp, "allocation.cpp")
	binary := filepath.Join(temp, "allocation")
	program := `
#include <array>
#include <cstddef>
#include "gpupow_v1_chunks.cuh"

int main() {
    using namespace sudharma::gpupowv1;
    std::array<void*, kProductionChunkCount> chunks{};
    unsigned allocations = 0;
    unsigned releases = 0;
    auto fail_fourth = [&](void** output, std::size_t bytes) {
        if (bytes != kProductionChunkBytes) return false;
        ++allocations;
        if (allocations == 4u) return false;
        *output = reinterpret_cast<void*>(static_cast<std::uintptr_t>(allocations));
        return true;
    };
    auto release = [&](void* value) {
        if (value != nullptr) ++releases;
    };
    if (allocate_dataset_chunks(&chunks, fail_fourth, release)) return 2;
    if (allocations != 4u || releases != 3u) return 3;
    for (void* chunk : chunks) if (chunk != nullptr) return 4;

    allocations = 0;
    releases = 0;
    auto succeed = [&](void** output, std::size_t bytes) {
        if (bytes != kProductionChunkBytes) return false;
        ++allocations;
        *output = reinterpret_cast<void*>(static_cast<std::uintptr_t>(allocations));
        return true;
    };
    if (!allocate_dataset_chunks(&chunks, succeed, release)) return 5;
    if (allocations != kProductionChunkCount || releases != 0u) return 6;
    release_dataset_chunks(&chunks, release);
    if (releases != kProductionChunkCount) return 7;
    for (void* chunk : chunks) if (chunk != nullptr) return 8;
    return 0;
}
`
	if err := os.WriteFile(source, []byte(program), 0o600); err != nil {
		t.Fatal(err)
	}
	include := filepath.Join("..", "cuda")
	cmd := exec.Command(compiler, "-std=c++17", "-O2", "-I", include, source, "-o", binary)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile CUDA chunk allocation contract: %v\n%s", err, out)
	}
	if out, err := exec.Command(binary).CombinedOutput(); err != nil {
		t.Fatalf("run CUDA chunk allocation contract: %v\n%s", err, out)
	}
}
