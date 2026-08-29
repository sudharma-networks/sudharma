package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestOpenCLProductionVectorSelfTestUsesDeviceKernel(t *testing.T) {
	kernel, err := os.ReadFile("../opencl/gpupow_v1_production_vectors.cl")
	if err != nil {
		t.Fatal(err)
	}
	host, err := os.ReadFile("../opencl/khushi_production_vectors_opencl.cpp")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"khushi_production_vectors",
		"khushi_dataset_item",
	} {
		if !strings.Contains(string(kernel), want) {
			t.Fatalf("OpenCL production vector kernel missing token %q", want)
		}
	}
	for _, want := range []string{
		"CL_DEVICE_TYPE_GPU",
		"kProductionCacheBytes",
		"262144",
		"4194303",
		"4194304",
		"33554431",
		"production-vector-self-test=ok",
		"clEnqueueNDRangeKernel",
		"clEnqueueReadBuffer",
	} {
		if !strings.Contains(string(host), want) {
			t.Fatalf("OpenCL production vector executable missing token %q", want)
		}
	}
}
