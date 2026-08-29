package gpupowv1

import "testing"

func TestDatasetItemLocationChunkBoundaries(t *testing.T) {
	cases := []struct {
		index  uint64
		chunk  uint32
		offset uint64
	}{
		{index: 0, chunk: 0, offset: 0},
		{index: 4194303, chunk: 0, offset: 268435392},
		{index: 4194304, chunk: 1, offset: 0},
		{index: 33554431, chunk: 7, offset: 268435392},
	}

	for _, tc := range cases {
		got, err := GPUV1ProductionMemory.DatasetItemLocation(tc.index)
		if err != nil || got.Chunk != tc.chunk || got.Offset != tc.offset {
			t.Fatalf("index=%d got=%+v err=%v", tc.index, got, err)
		}
	}
}

func TestDatasetItemLocationRejectsInvalidIndicesAndLayouts(t *testing.T) {
	for _, index := range []uint64{33554432, ^uint64(0)} {
		if _, err := GPUV1ProductionMemory.DatasetItemLocation(index); err == nil {
			t.Fatalf("out-of-range index accepted: %d", index)
		}
	}

	invalid := GPUV1ProductionMemory
	invalid.ItemBytes = 0
	if _, err := invalid.DatasetItemLocation(0); err == nil {
		t.Fatal("invalid layout accepted")
	}
}
