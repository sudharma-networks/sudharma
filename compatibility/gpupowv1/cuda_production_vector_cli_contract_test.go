package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestCUDAProductionVectorSelfTestUsesDeviceKernel(t *testing.T) {
	dataset, err := os.ReadFile("../cuda/gpupow_v1_dataset.h")
	if err != nil {
		t.Fatal(err)
	}
	vectors, err := os.ReadFile("../cuda/khushi_production_vectors.cu")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(dataset), "dataset_item_from_cache") {
		t.Fatal("CUDA dataset header missing dynamic production cache primitive")
	}
	for _, want := range []string{
		"production_dataset_vector_kernel",
		"dataset_item_from_cache",
		"kProductionCacheBytes",
		"262144",
		"4194303",
		"4194304",
		"33554431",
		"production-vector-self-test=ok",
		"cudaMemcpyHostToDevice",
		"cudaMemcpyDeviceToHost",
	} {
		if !strings.Contains(string(vectors), want) {
			t.Fatalf("CUDA production vector executable missing token %q", want)
		}
	}
}
