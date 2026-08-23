package p2p

import "net"

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

func (n *Node) trackInboundHandshake(listener net.Listener, conn net.Conn) bool {
	if n == nil || listener == nil || conn == nil {
		return false
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	if n.listener != listener {
		return false
	}
	if n.inboundHandshakeConns == nil {
		n.inboundHandshakeConns = make(map[net.Conn]struct{})
	}
	n.inboundHandshakeConns[conn] = struct{}{}
	return true
}

func (n *Node) untrackInboundHandshake(conn net.Conn) {
	if n == nil || conn == nil {
		return
	}

	n.mu.Lock()
	delete(n.inboundHandshakeConns, conn)
	n.mu.Unlock()
}
