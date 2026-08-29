package operations

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

const (
	gpuActivationChainFilename         = "sudharma-chain.json"
	gpuActivationRecordFilename        = "sudharma-gpu-v1-activation.json"
	gpuActivationAbortManifestFilename = "gpu-activation-abort-manifest.json"
)

// GPUActivationAbortOptions identifies one stopped node and the exact
// activation decision the operator intends to abort.
type GPUActivationAbortOptions struct {
	DataDirectory            string
	EvidenceDirectory        string
	ExpectedActivationHeight uint64
}

// GPUActivationAbortEvidence is the durable audit record written before the
// live activation record is moved out of the node data directory.
type GPUActivationAbortEvidence struct {
	CreatedAtUTC           string `json:"created_at_utc"`
	ChainTipHeight         uint64 `json:"chain_tip_height"`
	ActivationHeight       uint64 `json:"activation_height"`
	ChainSHA256            string `json:"chain_sha256"`
	ActivationRecordSHA256 string `json:"activation_record_sha256"`
}

// AbortGPUActivation performs an offline, pre-boundary abort. It never alters
// chain/state data and moves the activation record into the evidence directory
// instead of deleting it.
func AbortGPUActivation(options GPUActivationAbortOptions) (*GPUActivationAbortEvidence, error) {
	if options.DataDirectory == "" {
		return nil, fmt.Errorf("data directory is required")
	}
	if options.EvidenceDirectory == "" {
		return nil, fmt.Errorf("evidence directory is required")
	}
	if options.ExpectedActivationHeight == params.GPUV1ActivationDisabled {
		return nil, fmt.Errorf("expected activation height must be finite")
	}

	lock, err := LockDataDirectory(options.DataDirectory)
	if err != nil {
		return nil, err
	}
	defer lock.Close()

	if _, err := os.Lstat(options.EvidenceDirectory); err == nil {
		return nil, fmt.Errorf("evidence destination already exists")
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect evidence destination: %w", err)
	}

	activationPath := filepath.Join(options.DataDirectory, gpuActivationRecordFilename)
	policy, exists, err := loadGPUActivation(activationPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("activation record does not exist")
	}
	if policy.GPUV1ActivationHeight != options.ExpectedActivationHeight {
		return nil, fmt.Errorf(
			"activation height is %d, expected %d",
			policy.GPUV1ActivationHeight,
			options.ExpectedActivationHeight,
		)
	}

	chainPath := filepath.Join(options.DataDirectory, gpuActivationChainFilename)
	chain, err := blockchain.LoadChainFromFile(chainPath)
	if err != nil {
		return nil, fmt.Errorf("load chain before activation abort: %w", err)
	}
	if err := ValidateGPUActivationAbort(chain.Height(), policy.GPUV1ActivationHeight); err != nil {
		return nil, err
	}

	chainData, err := os.ReadFile(chainPath)
	if err != nil {
		return nil, fmt.Errorf("read chain evidence: %w", err)
	}
	activationData, err := os.ReadFile(activationPath)
	if err != nil {
		return nil, fmt.Errorf("read activation evidence: %w", err)
	}
	chainHash := sha256.Sum256(chainData)
	activationHash := sha256.Sum256(activationData)
	evidence := &GPUActivationAbortEvidence{
		CreatedAtUTC:           time.Now().UTC().Format(time.RFC3339Nano),
		ChainTipHeight:         chain.Height(),
		ActivationHeight:       policy.GPUV1ActivationHeight,
		ChainSHA256:            hex.EncodeToString(chainHash[:]),
		ActivationRecordSHA256: hex.EncodeToString(activationHash[:]),
	}

	if err := os.Mkdir(options.EvidenceDirectory, 0700); err != nil {
		return nil, fmt.Errorf("create evidence destination: %w", err)
	}
	manifestData, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode activation abort evidence: %w", err)
	}
	manifestData = append(manifestData, '\n')
	manifestPath := filepath.Join(options.EvidenceDirectory, gpuActivationAbortManifestFilename)
	manifest, err := os.OpenFile(manifestPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("create activation abort evidence: %w", err)
	}
	if _, err := manifest.Write(manifestData); err != nil {
		manifest.Close()
		return nil, fmt.Errorf("write activation abort evidence: %w", err)
	}
	if err := manifest.Sync(); err != nil {
		manifest.Close()
		return nil, fmt.Errorf("sync activation abort evidence: %w", err)
	}
	if err := manifest.Close(); err != nil {
		return nil, fmt.Errorf("close activation abort evidence: %w", err)
	}

	preservedPath := filepath.Join(options.EvidenceDirectory, gpuActivationRecordFilename)
	if err := os.Rename(activationPath, preservedPath); err != nil {
		return nil, fmt.Errorf("preserve activation record before abort: %w", err)
	}
	if err := syncDirectory(options.EvidenceDirectory); err != nil {
		return nil, fmt.Errorf("sync activation abort evidence directory: %w", err)
	}
	if err := syncDirectory(options.DataDirectory); err != nil {
		return nil, fmt.Errorf("sync node data directory after abort: %w", err)
	}
	return evidence, nil
}
