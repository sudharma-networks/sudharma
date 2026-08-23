package p2p

import (
	"encoding/json"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestHandleGetBlocksSecurePenalizesMalformedRequest(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer clearNodePeerScorer(node)

	peer := &PeerConnection{Info: PeerInfo{NodeID: "peer-a"}}
	payload, err := json.Marshal(GetBlocksMessage{StartHeight: 1, Limit: 0})
	if err != nil {
		t.Fatal(err)
	}
	message := &Message{Type: MessageGetBlocks, Payload: payload}

	node.handleGetBlocksSecure(peer, message)
	if got := node.PeerScore("peer-a"); got != -PeerPenaltyMalformed {
		t.Fatalf("expected malformed sync request score %d, got %d", -PeerPenaltyMalformed, got)
	}
}

func TestHandleBlocksResponseSecurePenalizesUnsolicitedBatch(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer clearNodePeerScorer(node)

	peer := &PeerConnection{Info: PeerInfo{NodeID: "peer-a"}}
	payload, err := json.Marshal(BlocksMessage{Blocks: []*blockchain.Block{{Height: 1}}})
	if err != nil {
		t.Fatal(err)
	}

	node.handleBlocksResponseSecure(peer, &Message{Type: MessageBlocks, Payload: payload})
	if got := node.PeerScore("peer-a"); got != -PeerPenaltyProtocolAbuse {
		t.Fatalf("expected unsolicited response score %d, got %d", -PeerPenaltyProtocolAbuse, got)
	}
}
