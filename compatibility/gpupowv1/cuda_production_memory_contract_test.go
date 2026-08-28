package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestCUDAProductionMemorySelfTestUsesChunkedAllocation(t *testing.T) {
	source, err := os.ReadFile("../cuda/khushi_miner.cu")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)

	for _, want := range []string{
		"gpupow_v1_chunks.cuh",
		"run_production_memory_self_test",
		"--production-memory-self-test",
		"allocate_dataset_chunks",
		"release_dataset_chunks",
		"kProductionDatasetBytes",
		"kProductionCacheBytes",
		"kProductionRuntimeReserveBytes",
		"kMinimumDedicatedVRAMBytes",
		"cudaMalloc(production dataset chunk)",
		"production-memory-self-test=ok",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("CUDA production memory path missing %q", want)
		}
	}
}
