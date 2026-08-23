package p2p

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/blockchain"
)

// AcceptBlock validates and atomically commits a block
// received from another Sudharma Network peer.
func (n *Node) AcceptBlock(
	block *blockchain.Block,
) error {

	if block == nil {
		return fmt.Errorf(
			"block cannot be nil",
		)
	}

	if block.MinerAddress == "" {
		return fmt.Errorf(
			"block miner address cannot be empty",
		)
	}

	chain := n.Chain()

	if chain == nil {
		return fmt.Errorf(
			"blockchain is not attached",
		)
	}

	state := n.State()

	if state == nil {
		return fmt.Errorf(
			"blockchain state is not attached",
		)
	}

	previous := chain.Tip()

	if previous == nil {
		return fmt.Errorf(
			"chain tip is nil",
		)
	}

	if err :=
		blockchain.ValidateBlockBasic(
			block,
			previous,
		); err != nil {

		return fmt.Errorf(
			"block validation failed: %w",
			err,
		)
	}

	// Process all transactions + subsidy + fees
	// against temporary state.
	workingState :=
		state.Clone()

	if _, err :=
		blockchain.ProcessBlock(
			workingState,
			block,
			block.MinerAddress,
		); err != nil {

		return fmt.Errorf(
			"block state processing failed: %w",
			err,
		)
	}

	// Commit block first.
	if err :=
		chain.AddBlock(
			block,
		); err != nil {

		return fmt.Errorf(
			"failed to add block: %w",
			err,
		)
	}

	// Commit resulting account state.
	if err :=
		state.ReplaceWith(
			workingState,
		); err != nil {

		return fmt.Errorf(
			"failed to commit state: %w",
			err,
		)
	}

	// Confirmed transactions no longer belong
	// in the local mempool.
	for _, tx := range block.Transactions {

		if tx == nil {
			continue
		}

		n.mempool.RemoveTransaction(
			tx.ID,
		)
	}

	n.RefreshChainStatus()

	return nil
}

func (n *Node) BroadcastBlock(
	block *blockchain.Block,
) error {

	if block == nil {
		return fmt.Errorf(
			"block cannot be nil",
		)
	}

	if block.MinerAddress == "" {
		return fmt.Errorf(
			"cannot broadcast block without miner address",
		)
	}

	data, err :=
		NewBlockMessage(
			block,
		)

	if err != nil {
		return err
	}

	n.mu.RLock()

	peers :=
		make(
			[]*PeerConnection,
			0,
			len(n.peers),
		)

	for _, peer := range n.peers {

		peers =
			append(
				peers,
				peer,
			)
	}

	n.mu.RUnlock()

	for _, peer := range peers {

		if err :=
			peer.write(data); err != nil {

			return fmt.Errorf(
				"failed to broadcast block to %s: %w",
				peer.Info.NodeID,
				err,
			)
		}
	}

	return nil
}
