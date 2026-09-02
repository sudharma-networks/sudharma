package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestCUDABenchmarkRunsSelectedProfileForFullRequestedDuration(t *testing.T) {
	raw, err := os.ReadFile("../cuda/khushi_miner.cu")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"kAutotuneCandidateMilliseconds",
		"final_started",
		"final_deadline = final_started + std::chrono::seconds(seconds)",
		"selected-profile-benchmark backend=cuda",
		"requested_seconds=%u",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("CUDA benchmark must tune briefly then run selected profile for the full requested duration; missing %q", want)
		}
	}
}

func TestOpenCLBenchmarkRunsSelectedProfileForFullRequestedDuration(t *testing.T) {
	raw, err := os.ReadFile("../opencl/khushi_miner_opencl.cpp")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"kAutotuneCandidateMilliseconds",
		"final_started",
		"final_deadline = final_started + std::chrono::seconds(seconds)",
		"selected-profile-benchmark backend=opencl",
		"requested_seconds=%u",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("OpenCL benchmark must tune briefly then run selected profile for the full requested duration; missing %q", want)
		}
	}
}

func TestOpenCLBenchmarkBoundsCandidatesByCompiledKernelLimit(t *testing.T) {
	raw, err := os.ReadFile("../opencl/khushi_miner_opencl.cpp")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"clGetKernelWorkGroupInfo",
		"CL_KERNEL_WORK_GROUP_SIZE",
		"kernel_max_work_group",
		"safe_max_local",
		"tuning::candidates(profile, safe_max_local)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("OpenCL benchmark must bound autotune candidates by the compiled khushi_search kernel limit; missing %q", want)
		}
	}
}
