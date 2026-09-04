package pow

import (
	"bytes"
	"testing"
)

func TestGPUV1CacheDeterministic(t *testing.T) {
	seed := GPUV1EpochSeed(0)
	a := GPUV1BuildCache(seed, 8)
	b := GPUV1BuildCache(seed, 8)
	if len(a) != 8 || len(b) != 8 {
		t.Fatalf("cache length = %d/%d want 8", len(a), len(b))
	}
	for i := range a {
		if !bytes.Equal(a[i][:], b[i][:]) {
			t.Fatalf("cache item %d is not deterministic", i)
		}
	}
}

func TestGPUV1CacheSeparatedByEpoch(t *testing.T) {
	a := GPUV1BuildCache(GPUV1EpochSeed(0), 8)
	b := GPUV1BuildCache(GPUV1EpochSeed(1), 8)
	if bytes.Equal(a[0][:], b[0][:]) {
		t.Fatal("different epoch seeds produced identical first cache item")
	}
}

func TestGPUV1CacheNodeWidthMatchesContract(t *testing.T) {
	cache := GPUV1BuildCache(GPUV1EpochSeed(0), 1)
	if len(cache) != 1 {
		t.Fatalf("cache length = %d want 1", len(cache))
	}
	if len(cache[0]) != int(GPUV1CacheNodeBytes) {
		t.Fatalf("cache node width = %d want %d", len(cache[0]), GPUV1CacheNodeBytes)
	}
}

func TestGPUV1CacheZeroSizeFailsClosed(t *testing.T) {
	cache := GPUV1BuildCache(GPUV1EpochSeed(0), 0)
	if len(cache) != 0 {
		t.Fatalf("zero-size cache length = %d", len(cache))
	}
}
