package operations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

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
	policy, exists, err := loadGPUActivation(path)
	if err != nil {
		return GPUActivationPolicy{}, err
	}
	if exists {
		if policy.GPUV1ActivationHeight == params.GPUV1ActivationDisabled {
			return GPUActivationPolicy{}, fmt.Errorf("persisted GPU-PoW activation record cannot be disabled")
		}
		if policy.GPUV1ActivationHeight != requestedActivationHeight {
			return GPUActivationPolicy{}, fmt.Errorf(
				"persisted GPU-PoW activation height is %d, requested %d",
				policy.GPUV1ActivationHeight,
				requestedActivationHeight,
			)
		}
		if !verifierReady {
			return GPUActivationPolicy{}, fmt.Errorf("GPU-PoW verifier is not ready")
		}
		return policy, nil
	}

	policy = GPUActivationPolicy{GPUV1ActivationHeight: requestedActivationHeight}
	if requestedActivationHeight == params.GPUV1ActivationDisabled {
		return policy, nil
	}
	if !verifierReady {
		return GPUActivationPolicy{}, fmt.Errorf("GPU-PoW verifier is not ready")
	}
	if requestedActivationHeight <= currentHeight ||
		requestedActivationHeight-currentHeight < gpuV1MinimumActivationLead {
		return GPUActivationPolicy{}, fmt.Errorf(
			"GPU-PoW activation height %d must be at least %d blocks after current height %d",
			requestedActivationHeight,
			gpuV1MinimumActivationLead,
			currentHeight,
		)
	}
	if err := persistGPUActivation(path, policy); err != nil {
		return GPUActivationPolicy{}, err
	}
	return policy, nil
}

type gpuActivationRecord struct {
	GPUV1ActivationHeight *uint64 `json:"gpu_v1_activation_height"`
}

func loadGPUActivation(path string) (GPUActivationPolicy, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return GPUActivationPolicy{}, false, nil
	}
	if err != nil {
		return GPUActivationPolicy{}, false, fmt.Errorf("inspect GPU-PoW activation record: %w", err)
	}
	if !info.Mode().IsRegular() {
		return GPUActivationPolicy{}, false, fmt.Errorf("GPU-PoW activation record must be a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		return GPUActivationPolicy{}, false, fmt.Errorf(
			"GPU-PoW activation record permissions %o expose group/world access",
			info.Mode().Perm(),
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return GPUActivationPolicy{}, false, fmt.Errorf("read GPU-PoW activation record: %w", err)
	}
	var record gpuActivationRecord
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return GPUActivationPolicy{}, false, fmt.Errorf("decode GPU-PoW activation record: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return GPUActivationPolicy{}, false, fmt.Errorf("decode GPU-PoW activation record: trailing data")
	}
	if record.GPUV1ActivationHeight == nil {
		return GPUActivationPolicy{}, false, fmt.Errorf(
			"decode GPU-PoW activation record: gpu_v1_activation_height is required",
		)
	}
	return GPUActivationPolicy{GPUV1ActivationHeight: *record.GPUV1ActivationHeight}, true, nil
}

func persistGPUActivation(path string, policy GPUActivationPolicy) error {
	if path == "" {
		return fmt.Errorf("GPU-PoW activation record path is required")
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return fmt.Errorf("create GPU-PoW activation directory: %w", err)
	}

	data, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("encode GPU-PoW activation record: %w", err)
	}
	data = append(data, '\n')

	file, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary GPU-PoW activation record: %w", err)
	}
	temporaryPath := file.Name()
	cleanup := true
	defer func() {
		_ = file.Close()
		if cleanup {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := file.Chmod(0600); err != nil {
		return fmt.Errorf("secure temporary GPU-PoW activation record: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write GPU-PoW activation record: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync GPU-PoW activation record: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close GPU-PoW activation record: %w", err)
	}

	// A hard-link commit is atomic and cannot replace an already-persisted
	// activation record. This makes conflicting first-arm attempts fail closed.
	if err := os.Link(temporaryPath, path); err != nil {
		return fmt.Errorf("commit GPU-PoW activation record without overwrite: %w", err)
	}
	if err := syncDirectory(directory); err != nil {
		return err
	}
	if err := os.Remove(temporaryPath); err != nil {
		return fmt.Errorf("remove temporary GPU-PoW activation record: %w", err)
	}
	cleanup = false
	if err := syncDirectory(directory); err != nil {
		return err
	}
	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open GPU-PoW activation directory: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync GPU-PoW activation directory: %w", err)
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
