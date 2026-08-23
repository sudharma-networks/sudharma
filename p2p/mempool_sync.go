package p2p

import (
	"fmt"
	"sort"

	"github.com/sudharma-networks/sudharma/transactions"
)

// syncMempoolToPeer sends a deterministic snapshot of this node's
// currently pending transactions to one connected peer.
//
// The receiving node independently validates every transaction
// against its own current blockchain state before admission.
func (n *Node) syncMempoolToPeer(
	peer *PeerConnection,
) (int, error) {

	if n == nil {
		return 0, fmt.Errorf(
			"node cannot be nil",
		)
	}

	if peer == nil {
		return 0, fmt.Errorf(
			"peer cannot be nil",
		)
	}

	txs :=
		n.MempoolSnapshot()

	sent := 0

	for _, tx := range txs {

		if tx == nil {
			continue
		}

		data, err :=
			NewTransactionMessage(
				tx,
			)

		if err != nil {
			return sent,
				fmt.Errorf(
					"failed encoding transaction %s: %w",
					tx.ID,
					err,
				)
		}

		if err :=
			peer.write(
				data,
			); err != nil {

			return sent,
				fmt.Errorf(
					"failed sending transaction %s: %w",
					tx.ID,
					err,
				)
		}

		sent++
	}

	return sent, nil
}

// SyncMempoolWithPeer performs explicit post-chain-sync mempool exchange.
//
// IMPORTANT:
// Call this only AFTER SyncFromPeer has completed. This prevents pending
// transactions from being rejected simply because the receiving node has
// not downloaded the funding blocks/state they depend on yet.
//
// The method:
//  1. sends this node's current pending transactions to the peer;
//  2. asks the peer to send its current pending transactions back.
func (n *Node) SyncMempoolWithPeer(
	nodeID string,
) error {

	if n == nil {
		return fmt.Errorf(
			"node cannot be nil",
		)
	}

	if nodeID == "" {
		return fmt.Errorf(
			"peer node ID cannot be empty",
		)
	}

	n.mu.RLock()

	peer, ok :=
		n.peers[nodeID]

	n.mu.RUnlock()

	if !ok {
		return fmt.Errorf(
			"peer not found: %s",
			nodeID,
		)
	}

	sent, err :=
		n.syncMempoolToPeer(
			peer,
		)

	if err != nil {
		return fmt.Errorf(
			"failed sending local mempool to %s: %w",
			nodeID,
			err,
		)
	}

	fmt.Printf(
		"[MEMPOOL] Sent %d pending transaction(s) to %s after chain sync\n",
		sent,
		nodeID,
	)

	request, err :=
		NewGetMempoolMessage()

	if err != nil {
		return err
	}

	if err :=
		peer.write(
			request,
		); err != nil {

		return fmt.Errorf(
			"failed requesting mempool from %s: %w",
			nodeID,
			err,
		)
	}

	fmt.Printf(
		"[MEMPOOL] Requested pending transactions from %s after chain sync\n",
		nodeID,
	)

	return nil
}

// MempoolSnapshot returns a deterministic copy of pending transactions.
// It is useful for synchronization, diagnostics and future RPC/API work.
func (n *Node) MempoolSnapshot() []*transactions.Transaction {

	if n == nil {
		return nil
	}

	txs :=
		n.mempool.AllTransactions()

	sort.Slice(
		txs,
		func(i, j int) bool {

			if txs[i].From != txs[j].From {
				return txs[i].From <
					txs[j].From
			}

			if txs[i].Nonce != txs[j].Nonce {
				return txs[i].Nonce <
					txs[j].Nonce
			}

			return txs[i].ID <
				txs[j].ID
		},
	)

	return txs
}
