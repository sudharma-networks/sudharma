package stratumcompat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/pool/stratum"
	"github.com/sudharma-networks/sudharma/pool/stratum/server"
	"github.com/sudharma-networks/sudharma/pool/stratum/transport"
)

const (
	compatWallet           = "9ccdc094489874bed888ffe4bdf9b8298f4c5131"
	compatTargetHex        = "0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f"
	compatHeaderPrefix     = "aabbccdd"
	compatLaneValue uint32 = 0x01020304
)

type compatSource struct {
	mu          sync.Mutex
	work        stratum.Work
	submissions []stratum.Candidate
}

func newCompatSource() *compatSource {
	return &compatSource{work: stratum.Work{
		WorkID:          "loopback-work-1",
		Algorithm:       "sudharma-gpupow-v1",
		TargetHex:       compatTargetHex,
		HeaderPrefixHex: compatHeaderPrefix,
		RewardAddress:   compatWallet,
		Version:         2,
		Height:          7600,
		Difficulty:      11,
	}}
}

func (s *compatSource) CurrentWork(ctx context.Context, rewardAddress string) (stratum.Work, error) {
	if err := ctx.Err(); err != nil {
		return stratum.Work{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	work := s.work
	work.RewardAddress = rewardAddress
	return work, nil
}

func (s *compatSource) Submit(ctx context.Context, candidate stratum.Candidate) (stratum.SourceResult, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.Lock()
	s.submissions = append(s.submissions, candidate)
	s.mu.Unlock()
	return stratum.SourceAccepted, nil
}

func (s *compatSource) submitted() []stratum.Candidate {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]stratum.Candidate(nil), s.submissions...)
}

type compatVerifier struct {
	networkTarget [32]byte
}

func newCompatVerifier(t *testing.T) compatVerifier {
	t.Helper()
	decoded, err := hex.DecodeString(compatTargetHex)
	if err != nil {
		t.Fatal(err)
	}
	var target [32]byte
	if len(decoded) != len(target) {
		t.Fatalf("network target bytes = %d, want %d", len(decoded), len(target))
	}
	copy(target[:], decoded)
	return compatVerifier{networkTarget: target}
}

func (v compatVerifier) MeetsTarget(ctx context.Context, _ stratum.Work, nonce uint64, target [32]byte) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if target == v.networkTarget {
		return uint32(nonce) == 2, nil
	}
	return true, nil
}

type compatLane uint32

func (l compatLane) Acquire(string, string) (uint32, error) { return uint32(l), nil }
func (compatLane) Release(string, string)                   {}

func newCompatFactory(t *testing.T, source *compatSource, calls *atomic.Int32) transport.SessionFactory {
	t.Helper()
	verifier := newCompatVerifier(t)
	return func() (*stratum.Session, error) {
		if calls != nil {
			calls.Add(1)
		}
		return stratum.NewSession(
			bytes.NewReader(bytes.Repeat([]byte{0x44}, 16)),
			source,
			verifier,
			stratum.Config{ShareDifficulty: 4, MaxDuplicateShares: 16, LaneSource: compatLane(compatLaneValue)},
		)
	}
}

func startCompatServer(
	t *testing.T,
	listener net.Listener,
	factory transport.SessionFactory,
	serverConfig server.Config,
) (context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- server.ServeListener(ctx, listener, factory, transport.Config{}, serverConfig)
	}()
	return cancel, done
}

func stopCompatServer(t *testing.T, cancel context.CancelFunc, done <-chan error) {
	t.Helper()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ServeListener error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeListener did not stop after cancellation")
	}
}

func dialCompatTCP(t *testing.T, address string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp4", address, time.Second)
	if err != nil {
		t.Fatalf("dial loopback Stratum: %v", err)
	}
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}
	return conn
}

func writeRequest(t *testing.T, conn net.Conn, request string) {
	t.Helper()
	if _, err := fmt.Fprintf(conn, "%s\n", request); err != nil {
		t.Fatalf("write Stratum request: %v", err)
	}
}

func readJSONMessage(t *testing.T, reader *bufio.Reader) map[string]any {
	t.Helper()
	line, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read Stratum response: %v", err)
	}
	var message map[string]any
	if err := json.Unmarshal(line, &message); err != nil {
		t.Fatalf("decode Stratum response %q: %v", line, err)
	}
	return message
}

func requireResponseResult(t *testing.T, reader *bufio.Reader, id float64, want any) map[string]any {
	t.Helper()
	message := readJSONMessage(t, reader)
	if got := message["id"]; got != id {
		t.Fatalf("response id = %#v, want %#v", got, id)
	}
	if got := message["error"]; got != nil {
		t.Fatalf("response error = %#v, want nil", got)
	}
	if got := message["result"]; got != want {
		t.Fatalf("response result = %#v, want %#v", got, want)
	}
	return message
}

func requireWorkMessages(t *testing.T, reader *bufio.Reader) (string, uint32) {
	t.Helper()
	difficulty := readJSONMessage(t, reader)
	if difficulty["method"] != "mining.set_difficulty" {
		t.Fatalf("first work method = %#v, want mining.set_difficulty", difficulty["method"])
	}
	difficultyParams, ok := difficulty["params"].([]any)
	if !ok || len(difficultyParams) != 1 || difficultyParams[0] != float64(4) {
		t.Fatalf("difficulty params = %#v, want [4]", difficulty["params"])
	}

	notify := readJSONMessage(t, reader)
	if notify["method"] != "mining.notify" {
		t.Fatalf("second work method = %#v, want mining.notify", notify["method"])
	}
	params, ok := notify["params"].([]any)
	if !ok || len(params) != 10 {
		t.Fatalf("notify params = %#v, want 10 fields", notify["params"])
	}
	if params[1] != "sudharma-gpupow-v1" || params[2] != float64(7600) || params[3] != compatTargetHex || params[4] != compatHeaderPrefix || params[5] != compatWallet || params[6] != float64(2) || params[7] != float64(11) || params[8] != float64(compatLaneValue) || params[9] != true {
		t.Fatalf("unexpected mining.notify params: %#v", params)
	}
	jobID, ok := params[0].(string)
	if !ok || len(jobID) != 64 {
		t.Fatalf("job ID = %#v, want 64-character string", params[0])
	}
	if _, err := hex.DecodeString(jobID); err != nil {
		t.Fatalf("job ID is not hexadecimal: %q: %v", jobID, err)
	}
	return jobID, compatLaneValue
}
