package pow

import (
	"encoding/hex"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestGPUV1AlgorithmContract(t *testing.T) {
	if GPUV1AlgorithmID != "sudharma-gpupow-v1" {
		t.Fatalf("algorithm id = %q", GPUV1AlgorithmID)
	}
	if GPUV1AlgorithmID != params.ProductionMiningAlgorithm {
		t.Fatalf("pow algorithm id %q != params production id %q", GPUV1AlgorithmID, params.ProductionMiningAlgorithm)
	}
	if GPUV1EpochLength != 7500 {
		t.Fatalf("epoch length = %d", GPUV1EpochLength)
	}
}

func TestGPUV1ProductionVerifierCacheContract(t *testing.T) {
	if GPUV1CacheNodeBytes != 64 {
		t.Fatalf("cache node bytes = %d", GPUV1CacheNodeBytes)
	}
	if GPUV1ProductionCacheBytes != 16<<20 {
		t.Fatalf("production cache bytes = %d", GPUV1ProductionCacheBytes)
	}
	if GPUV1ProductionCacheNodes != 262144 {
		t.Fatalf("production cache nodes = %d", GPUV1ProductionCacheNodes)
	}
	if GPUV1ProductionCacheBytes/GPUV1CacheNodeBytes != uint64(GPUV1ProductionCacheNodes) {
		t.Fatal("production cache byte/node contract is inconsistent")
	}
}

func TestGPUV1EpochForHeight(t *testing.T) {
	tests := []struct {
		height uint64
		want   uint64
	}{
		{0, 0},
		{7499, 0},
		{7500, 1},
		{14999, 1},
		{15000, 2},
	}
	for _, tc := range tests {
		if got := GPUV1EpochForHeight(tc.height); got != tc.want {
			t.Fatalf("height %d: epoch %d want %d", tc.height, got, tc.want)
		}
	}
}

func TestGPUV1EpochSeedVectors(t *testing.T) {
	vectors := []struct {
		epoch uint64
		want  string
	}{
		{0, "a20ac91b092ff0f1ac89feddcf16ca2f47b030eb841d75bcc026cde751f9ed7a"},
		{1, "636b2c7d76642f3e31f2de46e9f42bf14ac548788d65f46ff3ea1098a66cae39"},
		{17, "5998a8331029dedf68e124f3d2f98c1134624421089683c1d389d1753823c08b"},
	}
	for _, tc := range vectors {
		got := GPUV1EpochSeed(tc.epoch)
		if hex.EncodeToString(got[:]) != tc.want {
			t.Fatalf("epoch %d seed = %x, want %s", tc.epoch, got, tc.want)
		}
	}
}

func TestGPUV1NetworkActivationRemainsDisabled(t *testing.T) {
	if params.GPUV1TestnetActivationHeight != params.GPUV1ActivationDisabled {
		t.Fatalf("testnet activation unexpectedly armed at %d", params.GPUV1TestnetActivationHeight)
	}
	if params.GPUV1MainnetActivationHeight != params.GPUV1ActivationDisabled {
		t.Fatalf("mainnet activation unexpectedly armed at %d", params.GPUV1MainnetActivationHeight)
	}
}
