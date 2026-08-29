package server

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/sudharma-networks/sudharma/pool/stratum/transport"
)

func ServeListener(
	ctx context.Context,
	listener net.Listener,
	factory transport.SessionFactory,
	transportConfig transport.Config,
	config Config,
) error {
	if listener == nil || factory == nil {
		return ErrInvalidConfig
	}
	normalized, err := normalizeConfig(config)
	if err != nil {
		return err
	}

	serveCtx, cancel := context.WithCancel(ctx)
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		<-serveCtx.Done()
		_ = listener.Close()
	}()

	tracker := newAdmission(normalized.maxConnections, normalized.maxConnectionsPerIP)
	var connections sync.WaitGroup
	defer func() {
		cancel()
		_ = listener.Close()
		connections.Wait()
		<-watchDone
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return fmt.Errorf("accept Stratum connection: %w", err)
		}
		if conn == nil {
			return fmt.Errorf("accept Stratum connection: %w", ErrInvalidConfig)
		}

		key := sourceKey(conn.RemoteAddr())
		if !tracker.Acquire(key) {
			_ = conn.Close()
			continue
		}

		connections.Add(1)
		go func(conn net.Conn, key string) {
			defer connections.Done()
			defer tracker.Release(key)
			_ = transport.ServeConn(serveCtx, conn, factory, transportConfig)
		}(conn, key)
	}
}
