package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestOpenCLStagingSearchMatchesControlledGate(t *testing.T) {
	data, err := os.ReadFile("../opencl/khushi_miner_opencl.cpp")
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
		"max_nonces = 65536",
		"CPU fallback prohibited",
		"network mining remains interoperability-gated",
	} {
		if !strings.Contains(source, token) {
			t.Fatalf("OpenCL staging search contract missing %q", token)
		}
	}
}
