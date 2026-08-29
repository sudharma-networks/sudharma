package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestCUDAStagingSearchIsExplicitAndNonProduction(t *testing.T) {
	data, err := os.ReadFile("../cuda/khushi_miner.cu")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, token := range []string{
		"--staging-search",
		"--header-prefix-hex",
		"--target-hex",
		"--height",
		"--cache-nodes",
		"staging-solution-nonce=",
		"staging-search=not-found",
		"staging search only supports height=0 cache_nodes=8",
		"network mining is gated",
	} {
		if !strings.Contains(source, token) {
			t.Fatalf("CUDA staging search contract missing %q", token)
		}
	}
}
