package p2p

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestNodeChainAttachment(t *testing.T) {
	chain :=
		blockchain.NewChain()

	node, err :=
		NewNode(
			"chain-test-node",
			"127.0.0.1:0",
			999,
			"wrong-tip",
		)

	if err != nil {
		t.Fatal(err)
	}

	if err := node.SetChain(
		chain,
	); err != nil {

		t.Fatalf(
			"failed to attach chain: %v",
			err,
		)
	}

	if node.Chain() != chain {
		t.Fatal(
			"node returned wrong blockchain",
		)
	}

	if node.Height != chain.Height() {
		t.Fatalf(
			"wrong node height: expected %d, got %d",
			chain.Height(),
			node.Height,
		)
	}

	if node.TipHash !=
		chain.Tip().Hash() {

		t.Fatal(
			"node tip hash was not synchronized",
		)
	}
}

func TestNilChainRejected(t *testing.T) {
	node, err :=
		NewNode(
			"nil-chain-test",
			"127.0.0.1:0",
			0,
			"",
		)

	if err != nil {
		t.Fatal(err)
	}

	if err := node.SetChain(nil); err == nil {
		t.Fatal(
			"nil blockchain was accepted",
		)
	}
}
