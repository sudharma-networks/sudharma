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

func TestOfflineMainnetChainFixtureDoesNotAuthorizeRuntimeLaunch(t *testing.T) {
	if params.MainnetLaunchAuthorized {
		t.Fatal("test requires mainnet launch to remain unauthorized")
	}

	chain, err := newChainFromGenesisForNetwork(
		params.NetworkMainnet,
		NewMainnetGenesisBlock(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if got := chain.Network(); got != params.NetworkMainnet {
		t.Fatalf("network = %q, want %q", got, params.NetworkMainnet)
	}

	policy, err := chain.MonetaryPolicy()
	if err != nil {
		t.Fatal(err)
	}
	if policy != params.MonetaryPolicyMainnet {
		t.Fatalf("policy = %d, want %d", policy, params.MonetaryPolicyMainnet)
	}

	if _, err := NewChainFor(params.NetworkMainnet); err == nil {
		t.Fatal("runtime mainnet chain construction must remain unauthorized")
	}
}
