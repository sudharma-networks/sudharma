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

func TestGPUV1CUDAProgrammaticLaneMixMatchesReference(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable CUDA compatibility compile test uses g++")
	}
	gpp, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ unavailable")
	}

	tempDir := t.TempDir()
	harness := filepath.Join(tempDir, "program_loop_harness.cpp")
	binary := filepath.Join(tempDir, "sudharma-gpupow-program-loop")
	header := filepath.Join("..", "compatibility", "cuda", "gpupow_v1_program_loop.h")

	source := `#include <array>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include "gpupow_v1_program_loop.h"

static bool decode_seed(const char* text, std::array<std::uint8_t, 32>* out) {
    if (std::strlen(text) != 64u) return false;
    auto nibble = [](char c) -> int {
        if (c >= '0' && c <= '9') return c - '0';
        if (c >= 'a' && c <= 'f') return c - 'a' + 10;
        if (c >= 'A' && c <= 'F') return c - 'A' + 10;
        return -1;
    };
    for (std::size_t i = 0; i < out->size(); ++i) {
        const int hi = nibble(text[i * 2u]);
        const int lo = nibble(text[i * 2u + 1u]);
        if (hi < 0 || lo < 0) return false;
        (*out)[i] = static_cast<std::uint8_t>((hi << 4) | lo);
    }
    return true;
}

int main(int argc, char** argv) {
    if (argc != 4) return 64;
    char* end = nullptr;
    const unsigned long long work_seed = std::strtoull(argv[1], &end, 16);
    if (end == argv[1] || *end != '\0') return 65;
    end = nullptr;
    const unsigned long lane = std::strtoul(argv[2], &end, 10);
    if (end == argv[2] || *end != '\0' || lane > 0xfffffffful) return 66;

    std::array<std::uint8_t, 32> program_seed{};
    if (!decode_seed(argv[3], &program_seed)) return 67;

    std::array<std::array<std::uint8_t, 64>, 8> cache{};
    for (std::size_t node = 0; node < cache.size(); ++node) {
        for (std::size_t i = 0; i < cache[node].size(); ++i) {
            cache[node][i] = static_cast<std::uint8_t>((node * 29u + i * 7u + 3u) & 0xffu);
        }
    }

    const auto mix = sudharma::gpupowv1::programmatic_lane_mix(
        static_cast<std::uint64_t>(work_seed), static_cast<std::uint32_t>(lane), program_seed, cache);
    for (std::size_t i = 0; i < mix.size(); ++i) {
        if (i != 0u) std::fputc(',', stdout);
        std::printf("%08x", mix[i]);
    }
    std::fputc('\n', stdout);
    return 0;
}
`
	if err := os.WriteFile(harness, []byte(source), 0o600); err != nil {
		t.Fatalf("write CUDA program-loop harness: %v", err)
	}

	build := exec.Command(gpp, "-std=c++17", "-I"+filepath.Dir(header), harness, "-o", binary)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compile CUDA program-loop compatibility harness: %v\n%s", err, output)
	}

	cache := make([]GPUV1CacheNode, 8)
	for node := range cache {
		for i := range cache[node] {
			cache[node][i] = byte((node*29 + i*7 + 3) & 0xff)
		}
	}
	programSeed := GPUV1ProgramSeed(5)
	programSeedHex := hex.EncodeToString(programSeed[:])

	for _, tc := range []struct {
		workSeed uint64
		lane     uint32
	}{
		{workSeed: 0x0123456789abcdef, lane: 0},
		{workSeed: 0xfedcba9876543210, lane: 7},
		{workSeed: 0x1122334455667788, lane: 15},
	} {
		wantMix := gpuV1ProgrammaticLaneMix(tc.workSeed, tc.lane, programSeed, cache)
		wantParts := make([]string, len(wantMix))
		for i, word := range wantMix {
			wantParts[i] = fmt.Sprintf("%08x", word)
		}
		want := strings.Join(wantParts, ",")

		cmd := exec.Command(binary, fmt.Sprintf("%016x", tc.workSeed), fmt.Sprintf("%d", tc.lane), programSeedHex)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CUDA program-loop harness failed for seed=%016x lane=%d: %v\n%s", tc.workSeed, tc.lane, err, output)
		}
		if got := strings.TrimSpace(string(output)); got != want {
			t.Fatalf("CUDA programmatic lane mismatch for seed=%016x lane=%d:\n got %s\nwant %s", tc.workSeed, tc.lane, got, want)
		}
	}
}
