package p2p

const (
	// MaxConcurrentInboundHandshakes bounds unauthenticated inbound work before
	// a peer is admitted to the live peer set. It complements the established
	// peer cap and the handshake deadline so a connection flood cannot keep an
	// unbounded number of handshake goroutines alive.
	MaxConcurrentInboundHandshakes = 32
)

func (n *Node) tryAcquireInboundHandshake() bool {
	if n == nil || n.inboundHandshakeSlots == nil {
		return false
	}
	select {
	case n.inboundHandshakeSlots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (n *Node) releaseInboundHandshake() {
	if n == nil || n.inboundHandshakeSlots == nil {
		return
	}
	select {
	case <-n.inboundHandshakeSlots:
	default:
	}
}
