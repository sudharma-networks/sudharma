package p2p

import (
	"fmt"
	"net"
	"time"
)

const (
	// Established peers must exchange protocol traffic before this deadline. The
	// keepalive loop sends ping frames well before it expires, so quiet healthy
	// peers stay connected while protocol-stalled peers are still bounded.
	PeerReadIdleTimeout = 2 * time.Minute

	// TCP keepalive is a second transport-level safety net for half-open/dead
	// sockets beneath the application ping/pong protocol.
	PeerTCPKeepAliveIdle     = 30 * time.Second
	PeerTCPKeepAliveInterval = 15 * time.Second
	PeerTCPKeepAliveCount    = 4

	// PeerWriteTimeout limits one established-peer write operation.
	PeerWriteTimeout = 15 * time.Second
)

// setPeerReadDeadline refreshes the established-peer protocol idle deadline and
// enables bounded OS TCP keepalive where the connection is a real TCP socket.
func setPeerReadDeadline(conn net.Conn) error {
	if conn == nil {
		return fmt.Errorf("peer connection cannot be nil")
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			return fmt.Errorf("enable TCP keepalive: %w", err)
		}
		if err := tcpConn.SetKeepAliveConfig(net.KeepAliveConfig{
			Enable:   true,
			Idle:     PeerTCPKeepAliveIdle,
			Interval: PeerTCPKeepAliveInterval,
			Count:    PeerTCPKeepAliveCount,
		}); err != nil {
			return fmt.Errorf("configure TCP keepalive: %w", err)
		}
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
