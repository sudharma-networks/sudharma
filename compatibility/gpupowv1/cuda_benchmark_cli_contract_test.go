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
