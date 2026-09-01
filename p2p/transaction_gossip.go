package p2p

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/transactions"
)

// relayTransaction forwards an already validated and accepted
// transaction to every connected peer except the peer that
// originally sent it.
//
// IMPORTANT:
// This function must only be called AFTER the transaction has
// passed signature verification, blockchain-state validation,
// mempool validation, and has been successfully added locally.
func (n *Node) relayTransaction(
	tx *transactions.Transaction,
	excludeNodeID string,
) (int, error) {

	if tx == nil {
		return 0, fmt.Errorf(
			"transaction cannot be nil",
		)
	}

	if !tx.VerifyForNetwork(n.ActiveNetwork()) {
		return 0, fmt.Errorf(
			"cannot relay invalid transaction",
		)
	}

	data, err :=
		NewTransactionMessage(tx)

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

		if err := peer.write(data); err != nil {

			fmt.Printf(
				"[TX] Gossip to %s failed for %s: %v\n",
				peer.Info.NodeID,
				tx.ID,
				err,
			)

			continue
		}

		sent++
	}

	return sent, nil
}
