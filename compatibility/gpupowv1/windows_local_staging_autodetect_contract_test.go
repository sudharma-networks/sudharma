package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsLocalStagingGateAutoDetectsPackagedMiner(t *testing.T) {
	path := filepath.Join("..", "..", "scripts", "windows", "run-local-staging-gate.ps1")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read local staging gate: %v", err)
	}
	text := string(raw)
	for _, token := range []string{
		`[string]$MinerPath = ""`,
		`khushi-miner-nvidia.exe`,
		`khushi-miner-opencl.exe`,
		`packagedMinerCandidates`,
		`No packaged Khushi miner found`,
		`Multiple packaged Khushi miners found`,
		`selected_miner=`,
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("local staging gate missing packaged-miner autodetection token %q", token)
		}
	}
}

func TestPublicGuideShowsOneCommandPackagedLocalStagingGate(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "test-mining", "PUBLIC_TEST_MINING.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read public test-mining guide: %v", err)
	}
	text := string(raw)
	for _, token := range []string{
		`run-local-staging-gate.ps1 -Device 0 -BenchmarkSeconds 60`,
		`auto-detects the packaged CUDA or OpenCL miner`,
		`local-staging-gate=accepted`,
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("public test-mining guide missing one-command local-staging token %q", token)
		}
	}
}
