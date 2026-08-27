package pow

import "testing"

func TestGPUV1AlgorithmContract(t *testing.T) {
	if GPUV1AlgorithmID != "sudharma-gpupow-v1" {
		t.Fatalf("algorithm id = %q", GPUV1AlgorithmID)
	}
	if GPUV1EpochLength != 7500 {
		t.Fatalf("epoch length = %d", GPUV1EpochLength)
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

func TestGPUV1EpochSeedIsDeterministicAndSeparated(t *testing.T) {
	seed0 := GPUV1EpochSeed(0)
	seed0Again := GPUV1EpochSeed(0)
	seed1 := GPUV1EpochSeed(1)
	if seed0 != seed0Again {
		t.Fatal("epoch seed is not deterministic")
	}
	if seed0 == seed1 {
		t.Fatal("adjacent epochs must not share a seed")
	}
	if seed0 == [32]byte{} {
		t.Fatal("epoch zero seed must be Sudharma-domain-separated, not all zero")
	}
}
