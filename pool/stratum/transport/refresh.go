package transport

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/sudharma-networks/sudharma/pool/stratum"
)

type ticker interface {
	C() <-chan time.Time
	Stop()
}

type tickerFactory func(time.Duration) ticker

type realTicker struct {
	ticker *time.Ticker
}

func (t *realTicker) C() <-chan time.Time { return t.ticker.C }
func (t *realTicker) Stop()               { t.ticker.Stop() }

func newRealTicker(interval time.Duration) ticker {
	return &realTicker{ticker: time.NewTicker(interval)}
}

func runRefreshPump(
	ctx context.Context,
	conn net.Conn,
	session *stratum.Session,
	writer *messageWriter,
	interval time.Duration,
	newTicker tickerFactory,
	terminal chan<- error,
) {
	tick := newTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C():
			messages, err := session.RefreshWork(ctx)
			if err != nil {
				reportRefreshError(conn, terminal, fmt.Errorf("refresh Stratum work: %w", err))
				return
			}
			if err := writer.WriteMessages(messages); err != nil {
				reportRefreshError(conn, terminal, fmt.Errorf("write Stratum refresh: %w", err))
				return
			}
		}
	}
}

func reportRefreshError(conn net.Conn, terminal chan<- error, err error) {
	select {
	case terminal <- err:
	default:
	}
	_ = conn.SetReadDeadline(time.Now())
}
