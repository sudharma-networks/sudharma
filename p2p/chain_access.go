package p2p

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/blockchain"
)

// SetChain attaches the active Sudharma Network blockchain
// to this P2P node.
//
// Incoming blocks will be validated against this chain.
func (n *Node) SetChain(
	chain *blockchain.Chain,
) error {

	if chain == nil {
		return fmt.Errorf(
			"blockchain cannot be nil",
		)
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.chain = chain

	// Keep the advertised node status synchronized
	// with the actual blockchain.
	n.Height = chain.Height()

	tip := chain.Tip()

	if tip != nil {
		n.TipHash = tip.Hash()
	}

	return nil
}

// Chain returns the blockchain currently attached
// to this P2P node.
func (n *Node) Chain() *blockchain.Chain {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.chain
}

// RefreshChainStatus updates the height and tip hash
// advertised by this P2P node.
func (n *Node) RefreshChainStatus() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.chain == nil {
		return
	}

	n.Height = n.chain.Height()

	tip := n.chain.Tip()

	if tip != nil {
		n.TipHash = tip.Hash()
	}
}
