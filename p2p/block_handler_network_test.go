package p2p

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

func TestSetStateRejectsPolicyMismatchWithAttachedChain(t *testing.T) {
	ResetLocalNetworkIDForTests()
	t.Cleanup(ResetLocalNetworkIDForTests)

	chain := blockchain.NewChain()
	node, err := NewNode("policy-test", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := node.SetChain(chain); err != nil {
		t.Fatal(err)
	}

	wrongState := blockchain.NewStateFor(params.MonetaryPolicyMainnet)
	if err := node.SetState(wrongState); err == nil {
		t.Fatal("expected chain/state monetary policy mismatch to fail closed")
	}
}

func TestSetChainRejectsPolicyMismatchWithAttachedState(t *testing.T) {
	ResetLocalNetworkIDForTests()
	t.Cleanup(ResetLocalNetworkIDForTests)

	chain := blockchain.NewChain()
	node, err := NewNode("policy-test", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := node.SetState(blockchain.NewStateFor(params.MonetaryPolicyMainnet)); err != nil {
		t.Fatal(err)
	}
	if err := node.SetChain(chain); err == nil {
		t.Fatal("expected attached state policy mismatch to fail closed")
	}
}

func TestSetChainRejectsP2PNamespaceMismatch(t *testing.T) {
	SetLocalNetworkID(params.NetworkMainnet)
	t.Cleanup(ResetLocalNetworkIDForTests)

	chain := blockchain.NewChain()
	node, err := NewNode("network-test", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := node.SetChain(chain); err == nil {
		t.Fatal("expected P2P namespace/chain network mismatch to fail closed")
	}
}
