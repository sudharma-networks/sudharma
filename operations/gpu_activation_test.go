package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestGPUActivationAbsentConfigurationRemainsDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gpu-activation.json")
	policy, err := LoadOrPersistGPUActivation(
		path,
		params.GPUV1ActivationDisabled,
		0,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if policy.GPUV1ActivationHeight != params.GPUV1ActivationDisabled {
		t.Fatalf("activation height = %d, want disabled", policy.GPUV1ActivationHeight)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("disabled policy created activation record: %v", err)
	}
}

func TestGPUActivationRequiresLeadTimeAndReadyVerifier(t *testing.T) {
	tests := []struct {
		name       string
		activation uint64
		ready      bool
		wantError  bool
	}{
		{name: "less than 720", activation: 1719, ready: true, wantError: true},
		{name: "exactly 720", activation: 1720, ready: true},
		{name: "verifier not ready", activation: 1720, ready: false, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "gpu-activation.json")
			_, err := LoadOrPersistGPUActivation(
				path,
				test.activation,
				1000,
				test.ready,
			)
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError %t", err, test.wantError)
			}
		})
	}
}

func TestGPUActivationPersistsAtomicallyAndCannotChange(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "gpu-activation.json")
	policy, err := LoadOrPersistGPUActivation(path, 1720, 1000, true)
	if err != nil {
		t.Fatal(err)
	}
	if policy.GPUV1ActivationHeight != 1720 {
		t.Fatalf("activation height = %d, want 1720", policy.GPUV1ActivationHeight)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("activation record mode = %o, want 600", info.Mode().Perm())
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary activation record remains: %v", err)
	}

	if _, err := LoadOrPersistGPUActivation(path, 1720, 1719, true); err != nil {
		t.Fatalf("equal persisted activation rejected on restart: %v", err)
	}
	if _, err := LoadOrPersistGPUActivation(path, 1800, 1000, true); err == nil {
		t.Fatal("persisted activation height changed silently")
	}
	if _, err := LoadOrPersistGPUActivation(
		path,
		params.GPUV1ActivationDisabled,
		1000,
		true,
	); err == nil {
		t.Fatal("persisted activation cleared silently")
	}
}

func TestGPUActivationAbortValidation(t *testing.T) {
	if err := ValidateGPUActivationAbort(99, 100); err != nil {
		t.Fatalf("pre-boundary abort rejected: %v", err)
	}
	if err := ValidateGPUActivationAbort(100, 100); err == nil {
		t.Fatal("abort accepted at activation boundary")
	}
	if err := ValidateGPUActivationAbort(101, 100); err == nil {
		t.Fatal("abort accepted after activation boundary")
	}
}

func TestPersistGPUActivationDoesNotOverwriteExistingRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gpu-activation.json")
	existing := []byte("{\"gpu_v1_activation_height\":1720}\n")
	if err := os.WriteFile(path, existing, 0600); err != nil {
		t.Fatal(err)
	}
	if err := persistGPUActivation(path, GPUActivationPolicy{GPUV1ActivationHeight: 1800}); err == nil {
		t.Fatal("persistGPUActivation overwrote an existing activation record")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(existing) {
		t.Fatalf("existing activation record changed: %q", data)
	}
}

func TestPersistGPUActivationIgnoresStaleLegacyTemporaryFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gpu-activation.json")
	staleTemporaryPath := path + ".tmp"
	if err := os.WriteFile(staleTemporaryPath, []byte("stale partial record"), 0600); err != nil {
		t.Fatal(err)
	}

	policy := GPUActivationPolicy{GPUV1ActivationHeight: 1720}
	if err := persistGPUActivation(path, policy); err != nil {
		t.Fatalf("stale temporary file blocked activation persistence: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{\"gpu_v1_activation_height\":1720}\n" {
		t.Fatalf("persisted activation record = %q", data)
	}
}

func TestGPUActivationRejectsSymlinkedPersistedRecord(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target.json")
	path := filepath.Join(directory, "gpu-activation.json")
	if err := os.WriteFile(target, []byte("{\"gpu_v1_activation_height\":1720}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrPersistGPUActivation(path, 1720, 1000, true); err == nil {
		t.Fatal("symlinked persisted activation record accepted")
	}
}

func TestGPUActivationRejectsMissingHeightRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gpu-activation.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrPersistGPUActivation(path, 0, 0, true); err == nil {
		t.Fatal("activation record without gpu_v1_activation_height accepted")
	}
}

func TestGPUActivationRejectsUnknownRecordFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gpu-activation.json")
	if err := os.WriteFile(path, []byte("{\"gpu_v1_activation_height\":1720,\"unexpected\":true}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrPersistGPUActivation(path, 1720, 1000, true); err == nil {
		t.Fatal("activation record with unknown field accepted")
	}
}

func TestGPUActivationRejectsPersistedDisabledSentinel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gpu-activation.json")
	data := []byte(fmt.Sprintf("{\"gpu_v1_activation_height\":%d}\n", params.GPUV1ActivationDisabled))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrPersistGPUActivation(path, params.GPUV1ActivationDisabled, 0, true); err == nil {
		t.Fatal("persisted disabled sentinel accepted")
	}
}

func TestGPUActivationRejectsOverlyPermissivePersistedRecord(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not authoritative on Windows")
	}
	path := filepath.Join(t.TempDir(), "gpu-activation.json")
	if err := os.WriteFile(path, []byte("{\"gpu_v1_activation_height\":1720}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrPersistGPUActivation(path, 1720, 1000, true); err == nil {
		t.Fatal("overly permissive persisted activation record accepted")
	}
}

func TestGPUActivationPersistedRecordStillRequiresReadyVerifier(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gpu-activation.json")
	if err := os.WriteFile(path, []byte("{\"gpu_v1_activation_height\":1720}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrPersistGPUActivation(path, 1720, 1000, false); err == nil {
		t.Fatal("persisted activation accepted while verifier is not ready")
	}
}
