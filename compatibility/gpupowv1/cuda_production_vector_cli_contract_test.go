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
	miner, err := os.ReadFile("../cuda/khushi_miner.cu")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"production_dataset_vector_kernel",
		"dataset_item_from_cache",
	} {
		if !strings.Contains(string(dataset), want) {
			t.Fatalf("CUDA dataset header missing production vector token %q", want)
		}
	}
	for _, want := range []string{
		"--production-vector-self-test",
		"run_production_vector_self_test",
		"kProductionCacheBytes",
		"262144",
		"4194303",
		"4194304",
		"33554431",
		"production-vector-self-test=ok",
	} {
		if !strings.Contains(string(miner), want) {
			t.Fatalf("CUDA miner missing production vector token %q", want)
		}
	}
}
