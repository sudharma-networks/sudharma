package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestCUDABenchmarkRejectsMalformedDuration(t *testing.T) {
	source, err := os.ReadFile("../cuda/khushi_miner.cu")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, want := range []string{
		"invalid --benchmark seconds",
		"parse_u32(argv[arg + 1], &seconds)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("CUDA benchmark CLI must strictly validate duration; missing %q", want)
		}
	}
}

func TestCUDABenchmarkUsesRuntimeAutotuneProfiles(t *testing.T) {
	source, err := os.ReadFile("../cuda/khushi_miner.cu")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	start := strings.Index(text, "int run_benchmark(unsigned seconds)")
	end := strings.Index(text, "int run_staging_search(")
	if start < 0 || end <= start {
		t.Fatal("cannot isolate CUDA benchmark function")
	}
	benchmark := text[start:end]
	for _, want := range []string{
		"gpu_tuning_profile.h",
		"cuda_profile(prop.major, prop.minor)",
		"prop.maxThreadsPerBlock",
		"tuning::candidates",
		"tuning::work_items",
		"autotune-candidate",
		"autotune-selected",
		"blocks =",
	} {
		if want == "gpu_tuning_profile.h" {
			if !strings.Contains(text, want) {
				t.Fatalf("CUDA benchmark must include shared autotune policy; missing %q", want)
			}
			continue
		}
		if !strings.Contains(benchmark, want) {
			t.Fatalf("CUDA benchmark must autotune runtime launch geometry; missing %q", want)
		}
	}
	if strings.Contains(benchmark, "constexpr unsigned threads = 32u;\n    constexpr std::uint64_t nonces_per_launch = 32u;") {
		t.Fatal("CUDA benchmark must not retain the RTX-2060-era fixed 32-thread launch geometry")
	}
}
