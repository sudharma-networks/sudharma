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

	policy, err := chain.MonetaryPolicy()
	if err != nil {
		return fmt.Errorf("invalid blockchain network identity: %w", err)
	}
	if LocalNetworkID() != string(chain.Network()) {
		return fmt.Errorf(
			"P2P namespace mismatch: node=%q chain=%q",
			LocalNetworkID(),
			chain.Network(),
		)
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state != nil && n.state.MonetaryPolicy() != policy {
		return fmt.Errorf(
			"blockchain/state monetary policy mismatch: chain=%d state=%d",
			policy,
			n.state.MonetaryPolicy(),
		)
	}

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
