package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestAutomaticReorgAtMaximumDepthAllowed(t *testing.T) {
	err := ValidateAutomaticReorgDepth(params.MaxAutomaticReorgDepth)
	if err != nil {
		t.Fatalf("maximum allowed depth should be accepted: %v", err)
	}
}

func TestAutomaticReorgBeyondMaximumDepthRejected(t *testing.T) {
	err := ValidateAutomaticReorgDepth(params.MaxAutomaticReorgDepth + 1)
	if err == nil {
		t.Fatal("depth beyond maximum should be rejected")
	}
}

func TestAutomaticReorgZeroDepthAllowed(t *testing.T) {
	if err := ValidateAutomaticReorgDepth(0); err != nil {
		t.Fatalf("zero-depth reorg should be allowed: %v", err)
	}
}
