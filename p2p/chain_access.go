package p2p

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
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

// ActiveNetwork returns the network identity bound to the attached chain.
// When no chain is attached yet, the default public-testnet identity is used.
func (n *Node) ActiveNetwork() params.NetworkID {
	chain := n.Chain()
	if chain == nil {
		return params.DefaultNetwork
	}
	return chain.Network()
}

// AdvertisedChainStatus returns a synchronized snapshot of the chain status
// exposed to peers. Callers must use this instead of reading Height and TipHash
// directly while the node is running because RefreshChainStatus may update them
// from a peer read loop.
func (n *Node) AdvertisedChainStatus() (uint64, string) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Height, n.TipHash
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
