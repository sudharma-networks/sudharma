package p2p

import "fmt"

// handleGetBlocksSecure validates peer sync requests before serving them and
// applies reputation policy to malformed requests. Local serving/encoding
// failures are not blamed on the remote peer.
func (n *Node) handleGetBlocksSecure(peer *PeerConnection, message *Message) {
	request, err := DecodeGetBlocks(message)
	if err != nil {
		fmt.Printf("[SYNC] Invalid block request from %s: %v\n", peer.Info.NodeID, err)
		n.penalizePeerAndMaybeDisconnect(peer, PeerPenaltyMalformed, "malformed block sync request")
		return
	}

	blocks, err := n.blocksFromHeight(request.StartHeight, request.Limit)
	if err != nil {
		fmt.Printf("[SYNC] Failed serving blocks to %s: %v\n", peer.Info.NodeID, err)
		return
	}

	n.rewardValidPeerMessage(peer)
	if len(blocks) == 0 {
		return
	}

	data, err := NewBlocksMessage(blocks)
	if err != nil {
		fmt.Printf("[SYNC] Failed encoding blocks for %s: %v\n", peer.Info.NodeID, err)
		return
	}
	if err := peer.write(data); err != nil {
		fmt.Printf("[SYNC] Failed sending blocks to %s: %v\n", peer.Info.NodeID, err)
		return
	}

	fmt.Printf("[SYNC] Sent %d block(s) to %s starting at height %d\n", len(blocks), peer.Info.NodeID, request.StartHeight)
}

// handleBlocksResponseSecure rejects malformed or unsolicited block batches.
// RequestBlocks holds syncRequestMu for the full request/response window, so a
// successful TryLock here proves there is no active request awaiting a batch.
func (n *Node) handleBlocksResponseSecure(peer *PeerConnection, message *Message) {
	blocks, err := DecodeBlocks(message)
	if err != nil {
		fmt.Printf("[SYNC] Invalid blocks response from %s: %v\n", peer.Info.NodeID, err)
		n.penalizePeerAndMaybeDisconnect(peer, PeerPenaltyMalformed, "malformed blocks response")
		return
	}

	if n.syncRequestMu.TryLock() {
		n.syncRequestMu.Unlock()
		fmt.Printf("[SYNC] Unsolicited blocks response from %s ignored\n", peer.Info.NodeID)
		n.penalizePeerAndMaybeDisconnect(peer, PeerPenaltyProtocolAbuse, "unsolicited blocks response")
		return
	}

	select {
	case n.blocksResponse <- blocks:
		n.rewardValidPeerMessage(peer)
	default:
		fmt.Printf("[SYNC] Duplicate blocks response from %s ignored\n", peer.Info.NodeID)
		n.penalizePeerAndMaybeDisconnect(peer, PeerPenaltyProtocolAbuse, "duplicate blocks response")
	}
}
