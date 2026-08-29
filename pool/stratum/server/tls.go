package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

func prepareConn(ctx context.Context, conn net.Conn, config normalizedConfig) (net.Conn, error) {
	if config.tlsConfig == nil {
		return conn, nil
	}

	secure := tls.Server(conn, config.tlsConfig)
	if err := secure.SetDeadline(time.Now().Add(config.tlsHandshakeTimeout)); err != nil {
		return nil, fmt.Errorf("set Stratum TLS handshake deadline: %w", err)
	}
	if err := secure.HandshakeContext(ctx); err != nil {
		return nil, fmt.Errorf("handshake Stratum TLS: %w", err)
	}
	if err := secure.SetDeadline(time.Time{}); err != nil {
		return nil, fmt.Errorf("clear Stratum TLS handshake deadline: %w", err)
	}
	return secure, nil
}
