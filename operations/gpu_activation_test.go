package operations

import (
	"os"
	"path/filepath"
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
