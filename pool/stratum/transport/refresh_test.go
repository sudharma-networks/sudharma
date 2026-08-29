package transport

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/pool/stratum"
)

type manualTicker struct {
	c chan time.Time
}

func (t *manualTicker) C() <-chan time.Time { return t.c }
func (*manualTicker) Stop()                 {}

type refreshSource struct {
	mu   sync.Mutex
	work stratum.Work
	err  error
}

func (s *refreshSource) CurrentWork(context.Context, string) (stratum.Work, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.work, s.err
}

func (*refreshSource) Submit(context.Context, stratum.Candidate) (stratum.SourceResult, error) {
	return stratum.SourceAccepted, nil
}

func (s *refreshSource) set(work stratum.Work, err error) {
	s.mu.Lock()
	s.work = work
	s.err = err
	s.mu.Unlock()
}

func refreshFactory(source *refreshSource) SessionFactory {
	return func() (*stratum.Session, error) {
		return stratum.NewSession(
			bytes.NewReader(bytes.Repeat([]byte{0x33}, 16)),
			source,
			transportVerifier{},
			stratum.Config{ShareDifficulty: 4, LaneSource: transportLane(0x05060708)},
		)
	}
}

func refreshWork(id string, height uint64) stratum.Work {
	return stratum.Work{WorkID: id, Algorithm: "sudharma-gpupow-v1", TargetHex: "0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f", HeaderPrefixHex: "aabbccdd", RewardAddress: transportWallet, Version: 2, Height: height, Difficulty: 11}
}

func authorizeRefreshConnection(t *testing.T, client net.Conn) *bufio.Reader {
	t.Helper()
	reader := bufio.NewReader(client)
	if _, err := io.WriteString(client, "{\"id\":1,\"method\":\"mining.subscribe\",\"params\":[]}\n"); err != nil {
		t.Fatal(err)
	}
	_ = readTransportJSONLine(t, reader)
	if _, err := io.WriteString(client, "{\"id\":2,\"method\":\"mining.authorize\",\"params\":[\""+transportWallet+".rig_01\",\"x\"]}\n"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		_ = readTransportJSONLine(t, reader)
	}
	return reader
}

func TestServeConnPeriodicRefreshAndSingleTicker(t *testing.T) {
	server, client := net.Pipe()
	source := &refreshSource{work: refreshWork("refresh-work-1", 8000)}
	manual := &manualTicker{c: make(chan time.Time, 2)}
	var tickerCalls atomic.Int32
	tickerReady := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- serveConn(context.Background(), server, refreshFactory(source), Config{}, func(time.Duration) ticker {
			tickerCalls.Add(1)
			close(tickerReady)
			return manual
		})
	}()
	reader := authorizeRefreshConnection(t, client)
	select {
	case <-tickerReady:
	case <-time.After(time.Second):
		t.Fatal("refresh ticker did not start")
	}

	if _, err := io.WriteString(client, "{\"id\":3,\"method\":\"mining.authorize\",\"params\":[\""+transportWallet+".rig_01\",\"x\"]}\n"); err != nil {
		t.Fatal(err)
	}
	_ = readTransportJSONLine(t, reader)
	if got := tickerCalls.Load(); got != 1 {
		t.Fatalf("ticker factory calls = %d, want 1", got)
	}

	source.set(refreshWork("refresh-work-2", 8001), nil)
	manual.c <- time.Now()
	if method := readTransportJSONLine(t, reader)["method"]; method != "mining.set_difficulty" {
		t.Fatalf("first refresh method = %v", method)
	}
	if method := readTransportJSONLine(t, reader)["method"]; method != "mining.notify" {
		t.Fatalf("second refresh method = %v", method)
	}
	_ = client.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ServeConn error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeConn did not stop")
	}
}

func TestServeConnRefreshErrorWinsOverReadWakeup(t *testing.T) {
	server, client := net.Pipe()
	source := &refreshSource{work: refreshWork("refresh-error-1", 8100)}
	manual := &manualTicker{c: make(chan time.Time, 1)}
	done := make(chan error, 1)
	go func() {
		done <- serveConn(context.Background(), server, refreshFactory(source), Config{}, func(time.Duration) ticker { return manual })
	}()
	_ = authorizeRefreshConnection(t, client)
	want := errors.New("source unavailable")
	source.set(stratum.Work{}, want)
	manual.c <- time.Now()
	select {
	case err := <-done:
		if !errors.Is(err, want) {
			t.Fatalf("ServeConn error = %v, want source error", err)
		}
	case <-time.After(time.Second):
		t.Fatal("refresh error did not stop ServeConn")
	}
	_ = client.Close()
}
