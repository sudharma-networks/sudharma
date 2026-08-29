package main

import (
	"fmt"
	"os"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/operations"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/pow"
)

func loadOrCreateChainWithGPUActivation(
	chainPath string,
	activationPath string,
	configuredHeight uint64,
) (*blockchain.Chain, error) {
	if err := validateGPUStartupNetwork(params.GPUV1MainnetActivationHeight); err != nil {
		return nil, err
	}

	policy := blockchain.PoWPolicy{GPUV1ActivationHeight: configuredHeight}
	verifier, err := pow.NewChainProofVerifier(policy)
	if err != nil {
		return nil, fmt.Errorf("GPU-PoW verifier startup failed: %w", err)
	}
	if err := validateGPUStartupVerifier(verifier); err != nil {
		return nil, err
	}

	activationExists, err := regularActivationRecordExists(activationPath)
	if err != nil {
		return nil, fmt.Errorf("inspect GPU-PoW activation record: %w", err)
	}
	chainExists, err := storedChainExists(chainPath)
	if err != nil {
		return nil, fmt.Errorf("inspect blockchain storage: %w", err)
	}

	if activationExists {
		if _, err := operations.LoadOrPersistGPUActivation(
			activationPath,
			configuredHeight,
			0,
			true,
		); err != nil {
			return nil, fmt.Errorf("GPU-PoW activation startup failed: %w", err)
		}
	}

	tipForFirstArm := uint64(0)
	if chainExists && !activationExists {
		legacyChain, err := blockchain.LoadChainFromFile(chainPath)
		if err != nil {
			return nil, fmt.Errorf("inspect pre-activation chain tip: %w", err)
		}
		tipForFirstArm = legacyChain.Height()
	}

	var chain *blockchain.Chain
	if chainExists {
		chain, err = blockchain.LoadChainFromFileWithConsensus(chainPath, policy, verifier)
		if err != nil {
			return nil, fmt.Errorf("consensus chain replay failed: %w", err)
		}
	} else {
		chain, err = blockchain.NewChainWithConsensus(policy, verifier)
		if err != nil {
			return nil, fmt.Errorf("create consensus chain: %w", err)
		}
	}

	if !activationExists {
		if _, err := operations.LoadOrPersistGPUActivation(
			activationPath,
			configuredHeight,
			tipForFirstArm,
			true,
		); err != nil {
			return nil, fmt.Errorf("GPU-PoW activation startup failed: %w", err)
		}
	}

	if !chainExists {
		if err := chain.SaveToFile(chainPath); err != nil {
			return nil, fmt.Errorf("persist blockchain: %w", err)
		}
	}
	return chain, nil
}

func validateGPUStartupNetwork(mainnetHeight uint64) error {
	if mainnetHeight != params.GPUV1ActivationDisabled {
		return fmt.Errorf("mainnet GPU-PoW activation must remain disabled")
	}
	return nil
}

func validateGPUStartupVerifier(verifier blockchain.ProofVerifier) error {
	if verifier == nil || !verifier.SupportsVersion(2) {
		return fmt.Errorf("GPU-PoW verifier is not ready for Version 2")
	}
	return nil
}

func regularActivationRecordExists(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("GPU-PoW activation record must be a regular file")
	}
	return true, nil
}

func storedChainExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("blockchain storage must be a regular file")
	}
	return true, nil
}
