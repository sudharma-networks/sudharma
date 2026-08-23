package p2p

import (
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
)

// Mempool returns this node's active transaction mempool.
//
// This is used by the local miner to construct candidate blocks.
func (n *Node) Mempool() *mempool.Mempool {
	return n.mempool
}
