package blockchain

import (
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestCloneChainPreservesProofConfiguration(t *testing.T) {
	policy := PoWPolicy{GPUV1ActivationHeight: 100}
	verifier := &recordingProofVerifier{
		supported: map[uint32]bool{1: true, 2: true},
		result:    true,
	}
	source, err := NewChainForWithConsensus(params.NetworkPublicTestnet, policy, verifier)
	if err != nil {
		t.Fatal(err)
	}

	cloned, err := CloneChain(source)
	if err != nil {
		t.Fatal(err)
	}
	if got := cloned.PoWPolicy(); got != policy {
		t.Fatalf("cloned PoW policy = %+v, want %+v", got, policy)
	}
	_, clonedVerifier := cloned.proofValidationConfig()
	if clonedVerifier != verifier {
		t.Fatal("clone did not preserve configured proof verifier")
	}
}

func TestValidateAndCloneChainPreservesProofConfiguration(t *testing.T) {
	policy := PoWPolicy{GPUV1ActivationHeight: 100}
	verifier := &recordingProofVerifier{
		supported: map[uint32]bool{1: true, 2: true},
		result:    true,
	}
	source, err := NewChainForWithConsensus(params.NetworkPublicTestnet, policy, verifier)
	if err != nil {
		t.Fatal(err)
	}

	validated, err := ValidateAndCloneChain(source)
	if err != nil {
		t.Fatal(err)
	}
	if got := validated.PoWPolicy(); got != policy {
		t.Fatalf("validated clone PoW policy = %+v, want %+v", got, policy)
	}
	_, validatedVerifier := validated.proofValidationConfig()
	if validatedVerifier != verifier {
		t.Fatal("validated clone did not preserve configured proof verifier")
	}
}

func TestReplaceWithRejectsProofPolicyMismatch(t *testing.T) {
	verifier := &recordingProofVerifier{
		supported: map[uint32]bool{1: true, 2: true},
		result:    true,
	}
	currentPolicy := PoWPolicy{GPUV1ActivationHeight: 100}
	candidatePolicy := PoWPolicy{GPUV1ActivationHeight: 200}

	current, err := NewChainForWithConsensus(params.NetworkPublicTestnet, currentPolicy, verifier)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := NewChainForWithConsensus(params.NetworkPublicTestnet, candidatePolicy, verifier)
	if err != nil {
		t.Fatal(err)
	}

	err = current.ReplaceWith(candidate)
	if err == nil {
		t.Fatal("ReplaceWith accepted a candidate with a different proof-of-work policy")
	}
	if !strings.Contains(err.Error(), "proof-of-work policy mismatch") {
		t.Fatalf("unexpected replacement error: %v", err)
	}
	if got := current.PoWPolicy(); got != currentPolicy {
		t.Fatalf("current policy changed after rejected replacement: %+v", got)
	}
}
