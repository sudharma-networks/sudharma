package pow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGPUV1CUDAFinalDigestMatchesReference(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable CUDA compatibility compile test uses g++")
	}
	gpp, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ unavailable")
	}

	tempDir := t.TempDir()
	harness := filepath.Join(tempDir, "final_harness.cpp")
	binary := filepath.Join(tempDir, "sudharma-gpupow-final")
	header := filepath.Join("..", "compatibility", "cuda", "gpupow_v1_final.h")

	source := `#include <array>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include "gpupow_v1_final.h"

static int nibble(char c) {
    if (c >= '0' && c <= '9') return c - '0';
    if (c >= 'a' && c <= 'f') return c - 'a' + 10;
    if (c >= 'A' && c <= 'F') return c - 'A' + 10;
    return -1;
}

template <std::size_t N>
static bool parse_hex(const char* text, std::array<std::uint8_t, N>* out) {
    for (std::size_t i = 0; i < N; ++i) {
        const int hi = nibble(text[i * 2]);
        const int lo = nibble(text[i * 2 + 1]);
        if (hi < 0 || lo < 0) return false;
        (*out)[i] = static_cast<std::uint8_t>((hi << 4) | lo);
    }
    return text[N * 2] == '\0';
}

int main(int argc, char** argv) {
    if (argc != 3) return 64;
    std::array<std::uint8_t, 32> header_digest{};
    std::array<std::uint8_t, 32> program_seed{};
    if (!parse_hex(argv[1], &header_digest) || !parse_hex(argv[2], &program_seed)) return 65;

    std::array<std::array<std::uint8_t, 64>, 8> cache{};
    for (std::size_t node = 0; node < cache.size(); ++node) {
        for (std::size_t i = 0; i < cache[node].size(); ++i) {
            cache[node][i] = static_cast<std::uint8_t>((node * 29u + i * 7u + 3u) & 0xffu);
        }
    }

    const auto digest = sudharma::gpupowv1::final_digest_from_header(header_digest, program_seed, cache);
    for (const auto b : digest) std::printf("%02x", b);
    std::fputc('\n', stdout);
    return 0;
}
`
	if err := os.WriteFile(harness, []byte(source), 0o600); err != nil {
		t.Fatalf("write CUDA final-digest harness: %v", err)
	}

	build := exec.Command(gpp, "-std=c++17", "-O2", "-I"+filepath.Dir(header), harness, "-o", binary)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compile CUDA final-digest compatibility harness: %v\n%s", err, output)
	}

	cache := make([]GPUV1CacheNode, 8)
	for node := range cache {
		for i := range cache[node] {
			cache[node][i] = byte((node*29 + i*7 + 3) & 0xff)
		}
	}

	for i, tc := range []struct {
		headerLabel  string
		programIndex uint64
	}{
		{"khushi-final-vector-0", 0},
		{"khushi-final-vector-1", 1},
		{"khushi-final-vector-250", 250},
	} {
		headerDigest := sha256.Sum256([]byte(tc.headerLabel))
		programSeed := GPUV1ProgramSeed(tc.programIndex)
		workSeed := uint64(headerDigest[0]) |
			uint64(headerDigest[1])<<8 |
			uint64(headerDigest[2])<<16 |
			uint64(headerDigest[3])<<24 |
			uint64(headerDigest[4])<<32 |
			uint64(headerDigest[5])<<40 |
			uint64(headerDigest[6])<<48 |
			uint64(headerDigest[7])<<56
		mix := gpuV1ProgrammaticGroupDigest(workSeed, programSeed, cache)
		wantDigest := gpuV1FinalizeDigest(headerDigest, mix)
		want := hex.EncodeToString(wantDigest[:])

		cmd := exec.Command(binary, hex.EncodeToString(headerDigest[:]), hex.EncodeToString(programSeed[:]))
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CUDA final-digest harness failed for vector=%d: %v\n%s", i, err, output)
		}
		if got := strings.TrimSpace(string(output)); got != want {
			t.Fatalf("CUDA final digest mismatch for vector=%d: got %s want %s", i, got, want)
		}
	}

	_ = fmt.Sprintf
}
