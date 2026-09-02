package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func readHardwareTestContractFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

func TestHardwareTestWindowsBuildsRunOnDedicatedBranch(t *testing.T) {
	for _, path := range []string{
		"../../.github/workflows/khushi-windows-cuda.yml",
		"../../.github/workflows/khushi-windows-opencl.yml",
	} {
		text := readHardwareTestContractFile(t, path)
		if !strings.Contains(text, "feature/khushi-hardware-test-v0.2.0") {
			t.Fatalf("%s must build the dedicated hardware-test branch", path)
		}
	}
}

func TestHardwareTestPackageIsLocalhostOnlyAndOneClick(t *testing.T) {
	launcher := readHardwareTestContractFile(t, "../../scripts/windows/Run-GPU-Test.bat")
	for _, want := range []string{
		"BenchmarkSeconds 60",
		"run-local-staging-gate.ps1",
		"khushi-miner-nvidia.exe",
		"khushi-miner-opencl.exe",
		"127.0.0.1",
	} {
		if !strings.Contains(launcher, want) {
			t.Fatalf("one-click launcher missing %q", want)
		}
	}
	for _, forbidden := range []string{"--mine", "execute-api", "Seed-1", "Seed-2"} {
		if strings.Contains(launcher, forbidden) {
			t.Fatalf("one-click hardware gate must not contain %q", forbidden)
		}
	}
	readme := readHardwareTestContractFile(t, "../../docs/test-mining/KHUSHI_HARDWARE_TEST_README.txt")
	for _, want := range []string{"local-staging-gate=accepted", "does not activate", "evidence"} {
		if !strings.Contains(readme, want) {
			t.Fatalf("hardware-test README missing %q", want)
		}
	}
}

func TestHardwareTestReleaseUsesExactRevisionArtifactsAndImmutableTag(t *testing.T) {
	workflow := readHardwareTestContractFile(t, "../../.github/workflows/publish-khushi-hardware-test-v0.2.0.yml")
	for _, want := range []string{
		"khushi-hardware-test-v0.2.0",
		"feature/khushi-hardware-test-v0.2.0",
		"release(gpu): publish Khushi hardware test v0.2.0",
		"Khushi NVIDIA Windows CUDA",
		"Khushi Windows OpenCL",
		"head_sha=${GITHUB_SHA}",
		"source_revision=${GITHUB_SHA}",
		"khushi-miner-nvidia-windows",
		"khushi-miner-opencl-windows",
		"nvidia/khushi-miner-nvidia.exe",
		"opencl/khushi-miner-opencl.exe",
		"Run-GPU-Test.bat",
		"PACKAGE-METADATA.txt",
		"SHA256MANIFEST.txt",
		"RELEASE-SHA256SUMS.txt",
		"RELEASE-SOURCE-REVISION.txt",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("hardware-test publisher missing %q", want)
		}
	}
	for _, forbidden := range []string{"force=true", "--clobber", "git tag -f", "git push --force"} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("immutable hardware-test publisher must not contain %q", forbidden)
		}
	}
}

func TestHardwareTestReleaseFailsClosedOnExistingDifferentTag(t *testing.T) {
	workflow := readHardwareTestContractFile(t, "../../.github/workflows/publish-khushi-hardware-test-v0.2.0.yml")
	for _, want := range []string{
		"existing_tag_sha",
		"GITHUB_SHA",
		"refusing to move immutable tag",
		"branch_head",
		"refusing publication because branch head moved",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("publisher must fail closed on provenance drift; missing %q", want)
		}
	}
}
