package gpupowv1

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCUDAProductionDatasetBoundaryVectors(t *testing.T) {
	compiler, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ is not available")
	}

	raw, err := os.ReadFile("../../docs/gpu-pow-v1-production-memory-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture productionMemoryFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}

	cacheNodes := uint32(GPUV1ProductionMemory.CacheBytes / GPUV1ProductionMemory.ItemBytes)
	cache := buildCache(epochSeed(fixture.Epoch), cacheNodes)
	temp := t.TempDir()
	cachePath := filepath.Join(temp, "epoch0-cache.bin")
	cacheFile, err := os.Create(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	for i := range cache {
		if _, err := cacheFile.Write(cache[i][:]); err != nil {
			_ = cacheFile.Close()
			t.Fatal(err)
		}
	}
	if err := cacheFile.Close(); err != nil {
		t.Fatal(err)
	}

	source := filepath.Join(temp, "production_vectors.cpp")
	binary := filepath.Join(temp, "production_vectors")
	program := `
#include <cstdint>
#include <cstdio>
#include <fstream>
#include <vector>
#include "gpupow_v1_dataset.h"

int main(int argc, char** argv) {
    if (argc != 2) return 64;
    std::ifstream in(argv[1], std::ios::binary);
    if (!in) return 2;
    std::vector<std::uint8_t> cache((std::istreambuf_iterator<char>(in)), std::istreambuf_iterator<char>());
    constexpr std::uint32_t cache_nodes = 262144u;
    if (cache.size() != static_cast<std::size_t>(cache_nodes) * 64u) return 3;
    const std::uint32_t indices[] = {0u, 4194303u, 4194304u, 33554431u};
    for (std::uint32_t index : indices) {
        const auto item = sudharma::gpupowv1::dataset_item_from_cache(cache.data(), cache_nodes, index);
        std::printf("%u:", index);
        for (std::uint8_t value : item) std::printf("%02x", value);
        std::putchar('\n');
    }
    return 0;
}
`
	if err := os.WriteFile(source, []byte(program), 0o600); err != nil {
		t.Fatal(err)
	}
	include := filepath.Join("..", "cuda")
	cmd := exec.Command(compiler, "-std=c++17", "-O2", "-I", include, source, "-o", binary)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile CUDA production vector contract: %v\n%s", err, out)
	}
	out, err := exec.Command(binary, cachePath).CombinedOutput()
	if err != nil {
		t.Fatalf("run CUDA production vector contract: %v\n%s", err, out)
	}

	normalizedOutput := strings.ReplaceAll(string(out), "\r\n", "\n")
	lines := strings.Split(strings.TrimSpace(normalizedOutput), "\n")
	if len(lines) != len(fixture.Vectors) {
		t.Fatalf("CUDA vector line count=%d want %d\n%s", len(lines), len(fixture.Vectors), out)
	}
	for i, vector := range fixture.Vectors {
		want := strings.TrimSpace(strings.Join([]string{
			formatUint(vector.Index), vector.DigestHex,
		}, ":"))
		if lines[i] != want {
			t.Fatalf("CUDA production vector mismatch line %d:\ngot  %s\nwant %s", i, lines[i], want)
		}
	}
}

func formatUint(value uint64) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for value != 0 {
		i--
		buf[i] = digits[value%10]
		value /= 10
	}
	return string(buf[i:])
}
