package gpupowv1

import (
	"encoding/hex"
	"testing"
)

func TestGenerateProductionMemoryVectors(t *testing.T) {
	p := GPUV1ProductionMemory
	cache := buildCache(epochSeed(0), uint32(p.CacheBytes/p.ItemBytes))
	for _, index := range []uint64{0, 4194303, 4194304, 33554431} {
		location, err := p.DatasetItemLocation(index)
		if err != nil {
			t.Fatal(err)
		}
		item := datasetItem(cache, uint32(index))
		t.Logf("VECTOR index=%d chunk=%d offset=%d digest=%s", index, location.Chunk, location.Offset, hex.EncodeToString(item[:]))
	}
	t.Fatal("temporary production memory vector generator")
}
