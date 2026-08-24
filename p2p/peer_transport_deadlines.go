package p2p

import (
	"fmt"
	"net"
	"time"
)

const (
	// Established peers may legitimately be application-idle for long periods on
	// a quiet testnet. TCP keepalive, rather than an application read-idle
	// deadline, detects dead sockets without disconnecting healthy idle seeds.
	PeerTCPKeepAliveIdle     = 30 * time.Second
	PeerTCPKeepAliveInterval = 15 * time.Second
	PeerTCPKeepAliveCount    = 4

	// PeerWriteTimeout limits one established-peer write operation.
	PeerWriteTimeout = 15 * time.Second
)

// setPeerReadDeadline prepares an established connection for long-lived idle
// operation. Handshake work is already bounded separately. For real TCP
// sockets, enable OS TCP keepalive so dead peers are eventually detected while
// healthy quiet peers remain connected. Non-TCP transports used by tests simply
// have any prior read deadline cleared.
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
	return conn.SetReadDeadline(time.Time{})
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
