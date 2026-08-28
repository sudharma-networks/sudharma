package pow

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGPUV1CUDADatasetItemMatchesReference(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable CUDA compatibility compile test uses g++")
	}
	gpp, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ unavailable")
	}

	tempDir := t.TempDir()
	harness := filepath.Join(tempDir, "dataset_harness.cpp")
	binary := filepath.Join(tempDir, "sudharma-gpupow-dataset")
	header := filepath.Join("..", "compatibility", "cuda", "gpupow_v1_dataset.h")

	source := `#include <array>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include "gpupow_v1_dataset.h"

int main(int argc, char** argv) {
    if (argc != 2) return 64;
    char* end = nullptr;
    const unsigned long parsed = std::strtoul(argv[1], &end, 10);
    if (end == argv[1] || *end != '\0' || parsed > 0xfffffffful) return 65;

    std::array<std::array<std::uint8_t, 64>, 8> cache{};
    for (std::size_t node = 0; node < cache.size(); ++node) {
        for (std::size_t i = 0; i < cache[node].size(); ++i) {
            cache[node][i] = static_cast<std::uint8_t>((node * 29u + i * 7u + 3u) & 0xffu);
        }
    }

    const auto item = sudharma::gpupowv1::dataset_item(cache, static_cast<std::uint32_t>(parsed));
    for (const auto b : item) std::printf("%02x", b);
    std::fputc('\n', stdout);
    return 0;
}
`
	if err := os.WriteFile(harness, []byte(source), 0o600); err != nil {
		t.Fatalf("write CUDA dataset harness: %v", err)
	}

	build := exec.Command(gpp, "-std=c++17", "-I"+filepath.Dir(header), harness, "-o", binary)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compile CUDA dataset compatibility harness: %v\n%s", err, output)
	}

	cache := make([]GPUV1CacheNode, 8)
	for node := range cache {
		for i := range cache[node] {
			cache[node][i] = byte((node*29 + i*7 + 3) & 0xff)
		}
	}

	for _, index := range []uint32{0, 1, 7, 8, 0x12345678} {
		wantNode := GPUV1DatasetItem(cache, index)
		want := hex.EncodeToString(wantNode[:])
		cmd := exec.Command(binary, fmt.Sprintf("%d", index))
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CUDA dataset harness failed for index=%d: %v\n%s", index, err, output)
		}
		if got := strings.TrimSpace(string(output)); got != want {
			t.Fatalf("CUDA dataset item mismatch for index=%d: got %s want %s", index, got, want)
		}
	}
}
