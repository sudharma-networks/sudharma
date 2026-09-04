package pow

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func v022RepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve v0.2.2 contract source path")
	}
	return filepath.Dir(filepath.Dir(file))
}

func readV022File(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read required v0.2.2 file %s: %v", rel, err)
	}
	return string(data)
}

func requireV022Markers(t *testing.T, rel, content string, markers ...string) {
	t.Helper()
	for _, marker := range markers {
		if !strings.Contains(content, marker) {
			t.Errorf("%s missing required v0.2.2 marker %q", rel, marker)
		}
	}
}

func TestKhushiV022MainnetRehearsalReleaseContract(t *testing.T) {
	root := v022RepoRoot(t)

	required := map[string][]string{
		".github/workflows/khushi-hardware-test-v0.2.2.yml": {
			"workflow_dispatch:",
			"khushi-hardware-test-v0.2.2-windows",
			"khushi-hardware-test-v0.2.2",
			"source_revision=${GITHUB_SHA}",
			"rehearsal-blocks 50",
			"physical_evidence_gate=not_automatically_completed",
		},
		"cmd/sudharma-gpupow-staging/main.go": {
			"mainnet-rehearsal",
			"rehearsal-blocks",
			"NetworkMainnet",
			"GPUV1ProductionCacheNodes",
			"/v1/mining/staging/status",
		},
		"compatibility/cuda/khushi_miner.cu": {
			"262144u",
			"production-consensus-search",
		},
		"compatibility/opencl/khushi_miner_opencl.cpp": {
			"262144u",
			"production-consensus-search",
		},
		"scripts/windows/run-local-staging-gate.ps1": {
			"RehearsalBlocks",
			"-mainnet-rehearsal",
			"mainnet-rehearsal=accepted",
		},
		"scripts/windows/test-khushi-miner.ps1": {
			"RehearsalBlocks",
			"mainnet-rehearsal=accepted",
		},
		"scripts/windows/Run-GPU-Test.bat": {
			"Khushi Hardware Test v0.2.2",
			"-RehearsalBlocks 50",
			"khushi-hardware-test-launcher.log",
			"pause",
		},
	}

	for rel, markers := range required {
		content := readV022File(t, root, rel)
		requireV022Markers(t, rel, content, markers...)
	}

	for _, rel := range []string{
		"compatibility/cuda/khushi_miner.cu",
		"compatibility/opencl/khushi_miner_opencl.cpp",
	} {
		content := readV022File(t, root, rel)
		if strings.Contains(content, "staging search only supports height=0 cache_nodes=8") {
			t.Errorf("%s still contains compact-only staging restriction", rel)
		}
	}
}
