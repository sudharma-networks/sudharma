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

func TestGPUV1CUDACacheAccessMatchesReference(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable CUDA compatibility compile test uses g++")
	}
	gpp, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ unavailable")
	}

	tempDir := t.TempDir()
	harness := filepath.Join(tempDir, "memory_harness.cpp")
	binary := filepath.Join(tempDir, "sudharma-gpupow-memory")
	header := filepath.Join("..", "compatibility", "cuda", "gpupow_v1_memory.h")

	source := `#include <array>
#include <cstdint>
#include <cstdio>
#include "gpupow_v1_memory.h"

int main(int argc, char** argv) {
    if (argc != 3) return 64;
    unsigned selector = 0, word_index = 0;
    if (std::sscanf(argv[1], "%x", &selector) != 1 || std::sscanf(argv[2], "%u", &word_index) != 1) return 65;
    std::array<std::uint8_t, 64> node{};
    for (std::size_t i = 0; i < node.size(); ++i) node[i] = static_cast<std::uint8_t>(i * 3u + 1u);
    const auto cache_index = sudharma::gpupowv1::cache_index(selector, 8u);
    const auto word = sudharma::gpupowv1::word64(node, word_index);
    std::printf("cache-index=%u word=%08x\n", cache_index, word);
    return 0;
}
`
	if err := os.WriteFile(harness, []byte(source), 0o600); err != nil {
		t.Fatalf("write CUDA memory harness: %v", err)
	}

	build := exec.Command(gpp, "-std=c++17", "-I"+filepath.Dir(header), harness, "-o", binary)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compile CUDA memory compatibility harness: %v\n%s", err, output)
	}

	var node GPUV1CacheNode
	for i := range node {
		node[i] = byte(i*3 + 1)
	}
	vectors := []struct {
		selector uint32
		word     uint32
	}{
		{0x00000000, 0},
		{0x00000007, 1},
		{0x12345678, 7},
		{0xffffffff, 15},
	}
	for _, vector := range vectors {
		want := fmt.Sprintf("cache-index=%d word=%08x", vector.selector%8, gpuV1Word(node, vector.word))
		cmd := exec.Command(binary, fmt.Sprintf("%08x", vector.selector), fmt.Sprintf("%d", vector.word))
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CUDA memory harness failed for selector=%08x word=%d: %v\n%s", vector.selector, vector.word, err, output)
		}
		if got := strings.TrimSpace(string(output)); got != want {
			t.Fatalf("CUDA cache access mismatch for selector=%08x word=%d: got %s want %s", vector.selector, vector.word, got, want)
		}
	}
}
