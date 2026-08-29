package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/pool/stratum"
	"github.com/sudharma-networks/sudharma/pool/stratum/transport"
)

const serverWallet = "0123456789abcdef0123456789abcdef01234567"

type serverTestAddr string

func (a serverTestAddr) Network() string { return "test" }
func (a serverTestAddr) String() string  { return string(a) }

type acceptResult struct {
	conn net.Conn
	err  error
}

type scriptedListener struct {
	results chan acceptResult
	closed  chan struct{}
	once    sync.Once
}

func newScriptedListener(buffer int) *scriptedListener {
	return &scriptedListener{
		results: make(chan acceptResult, buffer),
		closed:  make(chan struct{}),
	}
}

func (l *scriptedListener) push(conn net.Conn, err error) {
	l.results <- acceptResult{conn: conn, err: err}
}

func (l *scriptedListener) Accept() (net.Conn, error) {
	select {
	case result := <-l.results:
		return result.conn, result.err
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *scriptedListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}

func (l *scriptedListener) Addr() net.Addr { return serverTestAddr("scripted-listener") }

type addressedConn struct {
	net.Conn
	remote net.Addr
}

func (c *addressedConn) RemoteAddr() net.Addr { return c.remote }

type serverSource struct {
	work stratum.Work
}

func (s *serverSource) CurrentWork(context.Context, string) (stratum.Work, error) {
	return s.work, nil
}

func (*serverSource) Submit(context.Context, stratum.Candidate) (stratum.SourceResult, error) {
	return stratum.SourceAccepted, nil
}

type serverVerifier struct{}

func (serverVerifier) MeetsTarget(context.Context, stratum.Work, uint64, [32]byte) (bool, error) {
	return false, nil
}

type serverLane uint32

func (l serverLane) Acquire(string, string) (uint32, error) { return uint32(l), nil }
func (serverLane) Release(string, string)                   {}

func newServerFactory(calls *atomic.Int32) transport.SessionFactory {
	source := &serverSource{work: stratum.Work{
		WorkID:          "server-work-1",
		Algorithm:       "sudharma-gpupow-v1",
		TargetHex:       "0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f",
		HeaderPrefixHex: "aabbccdd",
		RewardAddress:   serverWallet,
		Version:         2,
		Height:          7500,
		Difficulty:      11,
	}}
	return func() (*stratum.Session, error) {
		calls.Add(1)
		return stratum.NewSession(
			bytes.NewReader(bytes.Repeat([]byte{0x33}, 16)),
			source,
			serverVerifier{},
			stratum.Config{ShareDifficulty: 4, LaneSource: serverLane(0x01020304)},
		)
	}
}

func waitForServerCalls(t *testing.T, calls *atomic.Int32, want int32) {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if calls.Load() == want {
			return
		}
		select {
		case <-deadline.C:
			t.Fatalf("session factory calls = %d, want %d", calls.Load(), want)
		case <-ticker.C:
		}
	}
}

func readServerJSONLine(t *testing.T, reader *bufio.Reader) map[string]any {
	t.Helper()
	line, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read server line: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(line, &decoded); err != nil {
		t.Fatalf("decode server line %q: %v", line, err)
	}
	return decoded
}
