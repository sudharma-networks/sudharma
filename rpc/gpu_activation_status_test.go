package rpc

import (
	"math"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestGPUActivationStatusDisabledDoesNotExposeSentinel(t *testing.T) {
	status := deriveGPUActivationStatus(blockchain.LegacyOnlyPoWPolicy(), 0)
	if status.Phase != "disabled" {
		t.Fatalf("phase = %q, want disabled", status.Phase)
	}
	if status.ActivationHeight != nil {
		t.Fatalf("disabled activation height exposed: %d", *status.ActivationHeight)
	}
	if status.NextBlockVersion != 1 {
		t.Fatalf("next block version = %d, want 1", status.NextBlockVersion)
	}
}

func TestGPUActivationStatusArmedShowsBoundaryNextVersion(t *testing.T) {
	policy := blockchain.PoWPolicy{GPUV1ActivationHeight: 100}
	status := deriveGPUActivationStatus(policy, 50)
	if status.Phase != "armed" || status.ActivationHeight == nil || *status.ActivationHeight != 100 {
		t.Fatalf("unexpected armed status: %+v", status)
	}
	if status.NextBlockVersion != 1 {
		t.Fatalf("next block version = %d, want 1", status.NextBlockVersion)
	}

	boundary := deriveGPUActivationStatus(policy, 99)
	if boundary.Phase != "armed" || boundary.NextBlockVersion != 2 {
		t.Fatalf("unexpected boundary status: %+v", boundary)
	}
}

func TestGPUActivationStatusActiveUsesV2(t *testing.T) {
	status := deriveGPUActivationStatus(blockchain.PoWPolicy{GPUV1ActivationHeight: 100}, 100)
	if status.Phase != "active" || status.NextBlockVersion != 2 {
		t.Fatalf("unexpected active status: %+v", status)
	}
}

func TestGPUActivationStatusDoesNotOverflowNextHeight(t *testing.T) {
	status := deriveGPUActivationStatus(blockchain.PoWPolicy{GPUV1ActivationHeight: math.MaxUint64 - 1}, math.MaxUint64-2)
	if status.Phase != "armed" || status.NextBlockVersion != 2 {
		t.Fatalf("unexpected max-height boundary status: %+v", status)
	}
}
