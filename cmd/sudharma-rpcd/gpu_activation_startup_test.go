package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

type startupLegacyOnlyVerifier struct{}

func (startupLegacyOnlyVerifier) SupportsVersion(version uint32) bool { return version == 1 }
func (startupLegacyOnlyVerifier) Verify(block *blockchain.Block) bool { return block != nil }

func TestGPUStartupDisabledLeavesActivationUnarmed(t *testing.T) {
	directory := t.TempDir()
	chainPath := filepath.Join(directory, "chain.json")
	activationPath := filepath.Join(directory, "gpu-activation.json")

	chain, err := loadOrCreateChainWithGPUActivation(chainPath, activationPath, params.GPUV1ActivationDisabled)
	if err != nil {
		t.Fatal(err)
	}
	if chain.PoWPolicy().GPUV1ActivationHeight != params.GPUV1ActivationDisabled {
		t.Fatalf("activation = %d, want disabled", chain.PoWPolicy().GPUV1ActivationHeight)
	}
	if _, err := os.Stat(activationPath); !os.IsNotExist(err) {
		t.Fatalf("disabled startup created activation record: %v", err)
	}
}

func TestGPUStartupFirstArmHonors720BlockLead(t *testing.T) {
	directory := t.TempDir()
	chainPath := filepath.Join(directory, "chain.json")
	activationPath := filepath.Join(directory, "gpu-activation.json")

	if _, err := loadOrCreateChainWithGPUActivation(chainPath, activationPath, 719); err == nil {
		t.Fatal("accepted activation with less than 720 blocks of lead")
	}
	if _, err := os.Stat(activationPath); !os.IsNotExist(err) {
		t.Fatalf("failed first arm created activation record: %v", err)
	}
	chain, err := loadOrCreateChainWithGPUActivation(chainPath, activationPath, 720)
	if err != nil {
		t.Fatal(err)
	}
	if chain.PoWPolicy().GPUV1ActivationHeight != 720 {
		t.Fatalf("activation = %d, want 720", chain.PoWPolicy().GPUV1ActivationHeight)
	}
}

func TestGPUStartupRestartRequiresExactPersistedHeight(t *testing.T) {
	directory := t.TempDir()
	chainPath := filepath.Join(directory, "chain.json")
	activationPath := filepath.Join(directory, "gpu-activation.json")

	if _, err := loadOrCreateChainWithGPUActivation(chainPath, activationPath, 720); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOrCreateChainWithGPUActivation(chainPath, activationPath, 720); err != nil {
		t.Fatalf("equal restart failed: %v", err)
	}
	if _, err := loadOrCreateChainWithGPUActivation(chainPath, activationPath, 721); err == nil {
		t.Fatal("changed persisted activation height accepted")
	}
	if _, err := loadOrCreateChainWithGPUActivation(chainPath, activationPath, params.GPUV1ActivationDisabled); err == nil {
		t.Fatal("persisted activation silently cleared")
	}
}

func TestGPUStartupRefusesFiniteMainnetPolicy(t *testing.T) {
	if err := validateGPUStartupNetwork(720); err == nil {
		t.Fatal("finite mainnet activation accepted")
	}
	if err := validateGPUStartupNetwork(params.GPUV1ActivationDisabled); err != nil {
		t.Fatal(err)
	}
}

func TestGPUStartupRejectsVerifierWithoutVersion2(t *testing.T) {
	if err := validateGPUStartupVerifier(startupLegacyOnlyVerifier{}); err == nil {
		t.Fatal("startup accepted verifier without Version 2 support")
	}
	if err := validateGPUStartupVerifier(nil); err == nil {
		t.Fatal("startup accepted nil verifier")
	}
}

func TestGPUStartupRejectsSymlinkedActivationRecord(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target")
	activationPath := filepath.Join(directory, "gpu-activation.json")
	if err := os.WriteFile(target, []byte(`{"gpu_v1_activation_height":720}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, activationPath); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOrCreateChainWithGPUActivation(filepath.Join(directory, "chain.json"), activationPath, 720); err == nil {
		t.Fatal("symlinked activation record accepted")
	}
}

func TestGPUStartupRejectsMalformedActivationRecord(t *testing.T) {
	directory := t.TempDir()
	activationPath := filepath.Join(directory, "gpu-activation.json")
	if err := os.WriteFile(activationPath, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOrCreateChainWithGPUActivation(filepath.Join(directory, "chain.json"), activationPath, 720); err == nil {
		t.Fatal("malformed activation record accepted")
	}
}

func TestGPUStartupBadStoredChainDoesNotArm(t *testing.T) {
	directory := t.TempDir()
	chainPath := filepath.Join(directory, "chain.json")
	activationPath := filepath.Join(directory, "gpu-activation.json")
	if err := os.WriteFile(chainPath, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOrCreateChainWithGPUActivation(chainPath, activationPath, 720); err == nil {
		t.Fatal("corrupt stored chain accepted")
	}
	if _, err := os.Stat(activationPath); !os.IsNotExist(err) {
		t.Fatalf("failed startup created activation record: %v", err)
	}
}

func TestGPUStartupRejectsOverlyPermissiveActivationRecord(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not authoritative on Windows")
	}
	directory := t.TempDir()
	activationPath := filepath.Join(directory, "gpu-activation.json")
	if err := os.WriteFile(activationPath, []byte("{\"gpu_v1_activation_height\":720}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(activationPath, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOrCreateChainWithGPUActivation(filepath.Join(directory, "chain.json"), activationPath, 720); err == nil {
		t.Fatal("startup accepted an overly permissive activation record")
	}
}
