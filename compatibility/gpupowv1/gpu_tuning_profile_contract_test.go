package gpupowv1

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGPUTuningProfileCompilesAndRuns(t *testing.T) {
	compiler, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ is not installed in this environment")
	}
	output := filepath.Join(t.TempDir(), "gpu-tuning-profile-test")
	compile := exec.Command(
		compiler,
		"-std=c++17",
		"-Wall",
		"-Wextra",
		"-pedantic",
		"-I", "../gpu",
		"../gpu/gpu_tuning_profile_test.cpp",
		"-o", output,
	)
	if data, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile GPU tuning profile test: %v\n%s", err, data)
	}
	if data, err := exec.Command(output).CombinedOutput(); err != nil {
		t.Fatalf("run GPU tuning profile test: %v\n%s", err, data)
	}
}
