package p2p

import (
	"fmt"
	"net"
	"time"
)

const (
	// PeerReadIdleTimeout limits how long an established peer may remain silent
	// while a frame is being read. This prevents a stalled peer from holding a
	// goroutine and connection indefinitely.
	PeerReadIdleTimeout = 2 * time.Minute

	// PeerWriteTimeout limits one established-peer write operation.
	PeerWriteTimeout = 15 * time.Second
)

// setPeerReadDeadline refreshes the read deadline before receiving the next
// frame from an established peer.
func setPeerReadDeadline(conn net.Conn) error {
	if conn == nil {
		return fmt.Errorf("peer connection cannot be nil")
	}
	return conn.SetReadDeadline(time.Now().Add(PeerReadIdleTimeout))
}

// setPeerWriteDeadline bounds one write so a non-reading peer cannot block a
// sender indefinitely.
func setPeerWriteDeadline(conn net.Conn) error {
	if conn == nil {
		return fmt.Errorf("peer connection cannot be nil")
	}
	return conn.SetWriteDeadline(time.Now().Add(PeerWriteTimeout))
}

// clearPeerDeadlines clears transport deadlines after a bounded operation when
// a caller needs to return the connection to an unbounded state.
func clearPeerDeadlines(conn net.Conn) error {
	if conn == nil {
		return fmt.Errorf("peer connection cannot be nil")
	}
	return conn.SetDeadline(time.Time{})
}
