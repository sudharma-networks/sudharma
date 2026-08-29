package operations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sudharma-networks/sudharma/params"
)

const gpuV1MinimumActivationLead uint64 = 720

// GPUActivationPolicy is the operator-visible, persisted GPU-PoW activation
// decision. Once written, the activation height is immutable for that data
// directory so restarts cannot silently move or clear the boundary.
type GPUActivationPolicy struct {
	GPUV1ActivationHeight uint64 `json:"gpu_v1_activation_height"`
}

// LoadOrPersistGPUActivation loads an existing activation decision or, when
// explicitly armed for the first time, persists it atomically. The disabled
// sentinel never creates a record. A first activation requires verifier
// readiness and at least 720 blocks of lead time.
func LoadOrPersistGPUActivation(
	path string,
	requestedActivationHeight uint64,
	currentHeight uint64,
	verifierReady bool,
) (GPUActivationPolicy, error) {
	policy, err := loadGPUActivation(path)
	if err == nil {
		if policy.GPUV1ActivationHeight != requestedActivationHeight {
			return GPUActivationPolicy{}, fmt.Errorf(
				"persisted gpu_v1 activation height is %d, requested %d",
				policy.GPUV1ActivationHeight,
				requestedActivationHeight,
			)
		}
		return policy, nil
	}
	if !os.IsNotExist(err) {
		return GPUActivationPolicy{}, err
	}

	policy = GPUActivationPolicy{GPUV1ActivationHeight: requestedActivationHeight}
	if requestedActivationHeight == params.GPUV1ActivationDisabled {
		return policy, nil
	}
	if !verifierReady {
		return GPUActivationPolicy{}, fmt.Errorf("gpu_v1 verifier is not ready")
	}
	if requestedActivationHeight < currentHeight ||
		requestedActivationHeight-currentHeight < gpuV1MinimumActivationLead {
		return GPUActivationPolicy{}, fmt.Errorf(
			"gpu_v1 activation must be at least %d blocks ahead",
			gpuV1MinimumActivationLead,
		)
	}
	if path == "" {
		return GPUActivationPolicy{}, fmt.Errorf("gpu_v1 activation record path is required")
	}

	data, err := json.Marshal(policy)
	if err != nil {
		return GPUActivationPolicy{}, fmt.Errorf("encode gpu_v1 activation policy: %w", err)
	}
	data = append(data, '\n')

	temporaryPath := path + ".tmp"
	defer os.Remove(temporaryPath)
	file, err := os.OpenFile(temporaryPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return GPUActivationPolicy{}, fmt.Errorf("create gpu_v1 activation record: %w", err)
	}
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return GPUActivationPolicy{}, fmt.Errorf("secure gpu_v1 activation record: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return GPUActivationPolicy{}, fmt.Errorf("write gpu_v1 activation record: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return GPUActivationPolicy{}, fmt.Errorf("sync gpu_v1 activation record: %w", err)
	}
	if err := file.Close(); err != nil {
		return GPUActivationPolicy{}, fmt.Errorf("close gpu_v1 activation record: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return GPUActivationPolicy{}, fmt.Errorf("commit gpu_v1 activation record: %w", err)
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return GPUActivationPolicy{}, err
	}
	return policy, nil
}

func loadGPUActivation(path string) (GPUActivationPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return GPUActivationPolicy{}, err
	}
	var policy GPUActivationPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return GPUActivationPolicy{}, fmt.Errorf("decode gpu_v1 activation record: %w", err)
	}
	return policy, nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open gpu_v1 activation directory: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync gpu_v1 activation directory: %w", err)
	}
	return nil
}

// ValidateGPUActivationAbort permits rollback only before the persisted
// activation boundary has been reached.
func ValidateGPUActivationAbort(currentHeight, activationHeight uint64) error {
	if currentHeight >= activationHeight {
		return fmt.Errorf(
			"gpu_v1 activation cannot be aborted at height %d after boundary %d",
			currentHeight,
			activationHeight,
		)
	}
	return nil
}
