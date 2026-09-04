package pow

import (
	"bytes"
	"testing"
)

func TestGPUV1DatasetParentCount(t *testing.T) {
	if gpuV1DatasetParents != 512 {
		t.Fatalf("dataset parents = %d want 512", gpuV1DatasetParents)
	}
}

func TestGPUV1DatasetItemDeterministic(t *testing.T) {
	cache := GPUV1BuildCache(GPUV1EpochSeed(0), 64)
	a := GPUV1DatasetItem(cache, 0)
	b := GPUV1DatasetItem(cache, 0)
	if !bytes.Equal(a[:], b[:]) {
		t.Fatal("dataset item is not deterministic")
	}
	if a == (GPUV1CacheNode{}) {
		t.Fatal("dataset item must not be zero")
	}
}

func TestGPUV1DatasetItemDependsOnIndex(t *testing.T) {
	cache := GPUV1BuildCache(GPUV1EpochSeed(0), 64)
	a := GPUV1DatasetItem(cache, 0)
	b := GPUV1DatasetItem(cache, 1)
	if bytes.Equal(a[:], b[:]) {
		t.Fatal("different dataset indexes produced identical items")
	}
}

func TestGPUV1DatasetItemDependsOnEpochCache(t *testing.T) {
	cache0 := GPUV1BuildCache(GPUV1EpochSeed(0), 64)
	cache1 := GPUV1BuildCache(GPUV1EpochSeed(1), 64)
	a := GPUV1DatasetItem(cache0, 7)
	b := GPUV1DatasetItem(cache1, 7)
	if bytes.Equal(a[:], b[:]) {
		t.Fatal("different epoch caches produced identical dataset item")
	}
}

func TestGPUV1DatasetItemEmptyCacheFailsClosed(t *testing.T) {
	if got := GPUV1DatasetItem(nil, 0); got != (GPUV1CacheNode{}) {
		t.Fatal("empty cache must produce zero item")
	}
}
