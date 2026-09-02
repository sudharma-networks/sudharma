package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestOpenCLBenchmarkUsesRuntimeAutotuneProfiles(t *testing.T) {
	source, err := os.ReadFile("../opencl/khushi_miner_opencl.cpp")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, want := range []string{
		"gpu_tuning_profile.h",
		"CL_DEVICE_VENDOR",
		"CL_DEVICE_MAX_COMPUTE_UNITS",
		"CL_DEVICE_MAX_WORK_GROUP_SIZE",
		"CL_KERNEL_WORK_GROUP_SIZE",
		"clGetKernelWorkGroupInfo",
		"tuning::opencl_profile",
		"tuning::candidates",
		"tuning::work_items",
		"autotune-candidate",
		"autotune-selected",
		"&global, &local",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("OpenCL benchmark must autotune runtime launch geometry; missing %q", want)
		}
	}
	if strings.Contains(text, "std::size_t one = 1;\n        check(clEnqueueNDRangeKernel(rt.queue, kernel, 1, nullptr, &one, nullptr") {
		t.Fatal("OpenCL benchmark must not retain the single-work-item benchmark launch")
	}
}
