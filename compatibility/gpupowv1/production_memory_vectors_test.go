package gpupowv1

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

type productionMemoryFixture struct {
	Algorithm    string `json:"algorithm"`
	DatasetBytes uint64 `json:"dataset_bytes"`
	CacheBytes   uint64 `json:"cache_bytes"`
	ChunkBytes   uint64 `json:"chunk_bytes"`
	ItemBytes    uint64 `json:"item_bytes"`
	Epoch        uint64 `json:"epoch"`
	Vectors      []struct {
		Index     uint64 `json:"index"`
		Chunk     uint32 `json:"chunk"`
		Offset    uint64 `json:"offset"`
		DigestHex string `json:"digest_hex"`
	} `json:"vectors"`
}

func TestProductionMemoryVectorsMatchIndependentDataset(t *testing.T) {
	raw, err := os.ReadFile("../../docs/gpu-pow-v1-production-memory-vectors.json")
	if err != nil {
		t.Fatalf("read production memory fixture: %v", err)
	}

	var fixture productionMemoryFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("parse production memory fixture: %v", err)
	}
	p := GPUV1ProductionMemory
	if fixture.Algorithm != "sudharma-gpupow-v1" || fixture.DatasetBytes != p.DatasetBytes ||
		fixture.CacheBytes != p.CacheBytes || fixture.ChunkBytes != p.ChunkBytes ||
		fixture.ItemBytes != p.ItemBytes || fixture.Epoch != 0 {
		t.Fatalf("fixture policy mismatch: %+v", fixture)
	}
	if len(fixture.Vectors) != 4 {
		t.Fatalf("production memory vector count=%d want 4", len(fixture.Vectors))
	}

	wantIndices := map[uint64]bool{0: true, 4194303: true, 4194304: true, 33554431: true}
	seen := make(map[uint64]bool, len(wantIndices))
	cacheNodes := uint32(p.CacheBytes / p.ItemBytes)
	cache := buildCache(epochSeed(fixture.Epoch), cacheNodes)
	for _, vector := range fixture.Vectors {
		if !wantIndices[vector.Index] || seen[vector.Index] {
			t.Fatalf("unexpected or duplicate vector index: %d", vector.Index)
		}
		seen[vector.Index] = true
		location, err := p.DatasetItemLocation(vector.Index)
		if err != nil || location.Chunk != vector.Chunk || location.Offset != vector.Offset {
			t.Fatalf("index=%d location=%+v err=%v fixture=%d/%d", vector.Index, location, err, vector.Chunk, vector.Offset)
		}
		item := datasetItem(cache, uint32(vector.Index))
		if got := hex.EncodeToString(item[:]); got != vector.DigestHex {
			t.Fatalf("index=%d digest=%s want %s", vector.Index, got, vector.DigestHex)
		}
	}
}
