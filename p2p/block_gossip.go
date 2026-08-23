package p2p

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/blockchain"
)

// relayBlock forwards an already validated and accepted block
// to every connected peer except the peer that sent it.
//
// This function must only be called AFTER AcceptBlock succeeds.
func (n *Node) relayBlock(
	block *blockchain.Block,
	excludeNodeID string,
) (int, error) {

	if block == nil {
		return 0, fmt.Errorf(
			"block cannot be nil",
		)
	}

	if block.MinerAddress == "" {
		return 0, fmt.Errorf(
			"cannot relay block without miner address",
		)
	}

	data, err :=
		NewBlockMessage(block)

	if err != nil {
		return 0, err
	}

	n.mu.RLock()

	peers := make(
		[]*PeerConnection,
		0,
		len(n.peers),
	)

	for nodeID, peer := range n.peers {

		if nodeID == excludeNodeID {
			continue
		}

		peers = append(
			peers,
			peer,
		)
	}

	n.mu.RUnlock()

	sent := 0

	for _, peer := range peers {

		if err :=
			peer.write(data); err != nil {

			fmt.Printf(
				"[BLOCK] Gossip to %s failed for block #%d: %v\n",
				peer.Info.NodeID,
				block.Height,
				err,
			)

			continue
		}

		sent++
	}

	return sent, nil
}
