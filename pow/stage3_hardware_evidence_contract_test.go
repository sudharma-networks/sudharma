package pow

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func stage3RepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve stage3 contract source path")
	}
	return filepath.Dir(filepath.Dir(file))
}

func readStage3ContractFile(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read required Stage 3 evidence file %s: %v", rel, err)
	}
	return string(data)
}

func requireStage3Markers(t *testing.T, rel, content string, markers ...string) {
	t.Helper()
	for _, marker := range markers {
		if !strings.Contains(content, marker) {
			t.Errorf("%s missing required Stage 3 marker %q", rel, marker)
		}
	}
}

func TestStage3HardwareEvidenceBundleContract(t *testing.T) {
	root := stage3RepoRoot(t)

	required := map[string][]string{
		".github/workflows/khushi-hardware-test-v0.2.1.yml": {
			"workflow_dispatch:",
			"khushi-hardware-test-v0.2.1",
			"source_revision=${GITHUB_SHA}",
			"127.0.0.1:28646",
			"physical_evidence_gate=not_automatically_completed",
		},
		"cmd/sudharma-gpupow-staging/main.go": {
			"127.0.0.1:28646",
			"GPUV1ReferenceDigest",
			"GPUV1BuildCache",
			"/v1/mining/staging/challenge",
			"/v1/mining/staging/submit",
		},
		"compatibility/cuda/khushi_miner.cu": {
			"2a7c15fc6c84a67d43ff7074ac5835aa433145f89d10d1d9e36a99fe22da4b2b",
			"--vector-self-test",
			"--production-memory-self-test",
			"--staging-search",
		},
		"compatibility/opencl/khushi_miner_opencl.cpp": {
			"2a7c15fc6c84a67d43ff7074ac5835aa433145f89d10d1d9e36a99fe22da4b2b",
			"--vector-self-test",
			"--production-memory-self-test",
			"--staging-search",
		},
		"scripts/windows/Run-GPU-Test.bat": {
			"Khushi Hardware Test v0.2.1",
			"khushi-hardware-test-launcher.log",
			"run-local-staging-gate.ps1",
			"pause",
		},
		"scripts/windows/run-local-staging-gate.ps1": {
			"http://127.0.0.1:28646",
			"local-staging-gate=accepted",
			"consensus_activation=disabled",
		},
		"scripts/windows/test-khushi-miner.ps1": {
			"--vector-self-test",
			"--production-memory-self-test",
			"--benchmark",
			"--staging-search",
		},
	}

	for rel, markers := range required {
		content := readStage3ContractFile(t, root, rel)
		requireStage3Markers(t, rel, content, markers...)
	}

	for _, rel := range []string{
		"cmd/sudharma-gpupow-staging/main.go",
		"scripts/windows/run-local-staging-gate.ps1",
	} {
		content := readStage3ContractFile(t, root, rel)
		if strings.Contains(content, "0.0.0.0") {
			t.Errorf("%s must remain localhost-only; found wildcard bind", rel)
		}
	}
}
