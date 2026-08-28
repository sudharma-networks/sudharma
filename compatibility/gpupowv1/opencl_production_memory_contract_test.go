package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestOpenCLProductionMemorySelfTestUsesChunkedAllocation(t *testing.T) {
	source, err := os.ReadFile("../opencl/khushi_miner_opencl.cpp")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)

	for _, want := range []string{
		"--production-memory-self-test",
		"production_memory_self_test",
		"kProductionDatasetBytes",
		"kProductionCacheBytes",
		"kProductionRuntimeReserveBytes",
		"kProductionChunkBytes",
		"kProductionChunkCount",
		"kMinimumDedicatedVRAMBytes",
		"CL_DEVICE_MAX_MEM_ALLOC_SIZE",
		"production dataset chunk",
		"production-memory-self-test=ok",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("OpenCL production memory path missing %q", want)
		}
	}
}
