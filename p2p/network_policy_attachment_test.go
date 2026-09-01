package p2p

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

func TestSetChainRejectsP2PNamespaceMismatch(t *testing.T) {
	ResetLocalNetworkIDForTests()
	defer ResetLocalNetworkIDForTests()

	chain := blockchain.NewChain()
	node, err := NewNode("node-a", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}

	SetLocalNetworkID(params.NetworkMainnet)
	if err := node.SetChain(chain); err == nil {
		t.Fatal("expected SetChain to reject a P2P namespace mismatch")
	}
}

func TestSetChainRejectsAttachedStatePolicyMismatch(t *testing.T) {
	ResetLocalNetworkIDForTests()
	defer ResetLocalNetworkIDForTests()

	node, err := NewNode("node-a", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := node.SetState(blockchain.NewStateFor(params.MonetaryPolicyMainnet)); err != nil {
		t.Fatal(err)
	}

	if err := node.SetChain(blockchain.NewChain()); err == nil {
		t.Fatal("expected SetChain to reject an attached state with the wrong monetary policy")
	}
}

func TestSetStateRejectsAttachedChainPolicyMismatch(t *testing.T) {
	ResetLocalNetworkIDForTests()
	defer ResetLocalNetworkIDForTests()

	chain := blockchain.NewChain()
	node, err := NewNode("node-a", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := node.SetChain(chain); err != nil {
		t.Fatal(err)
	}

	if err := node.SetState(blockchain.NewStateFor(params.MonetaryPolicyMainnet)); err == nil {
		t.Fatal("expected SetState to reject a state with the wrong monetary policy")
	}
}
