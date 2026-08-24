package p2p

import (
	"sync/atomic"
	"time"
)

// PeerKeepaliveInterval is deliberately shorter than PeerReadIdleTimeout so
// quiet but healthy peers exchange application-level traffic before either
// read loop reaches its idle deadline.
const PeerKeepaliveInterval = PeerReadIdleTimeout / 3

var peerKeepaliveNonce atomic.Uint64

func (n *Node) keepPeerAlive(peer *PeerConnection) {
	n.keepPeerAliveWithInterval(peer, PeerKeepaliveInterval)
}

func (n *Node) keepPeerAliveWithInterval(peer *PeerConnection, interval time.Duration) {
	if n == nil || peer == nil || interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if !n.isCurrentPeerConnection(peer) {
			return
		}

		ping, err := NewPingMessage(peerKeepaliveNonce.Add(1))
		if err != nil {
			return
		}
		if err := peer.write(ping); err != nil {
			n.removePeerConnection(peer)
			if peer.conn != nil {
				_ = peer.conn.Close()
			}
			return
		}
	}
}

func (n *Node) isCurrentPeerConnection(peer *PeerConnection) bool {
	if n == nil || peer == nil || peer.Info.NodeID == "" {
		return false
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	current, ok := n.peers[peer.Info.NodeID]
	return ok && current == peer
}
