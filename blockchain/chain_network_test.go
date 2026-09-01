package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestNewChainBindsPublicTestnetNetwork(t *testing.T) {
	chain := NewChain()

	if got := chain.Network(); got != params.NetworkPublicTestnet {
		t.Fatalf("network = %q, want %q", got, params.NetworkPublicTestnet)
	}

	policy, err := chain.MonetaryPolicy()
	if err != nil {
		t.Fatal(err)
	}
	if policy != params.MonetaryPolicyPublicTestnet {
		t.Fatalf("policy = %d, want %d", policy, params.MonetaryPolicyPublicTestnet)
	}
}
