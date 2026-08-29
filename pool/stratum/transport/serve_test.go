package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/pool/stratum"
)

const transportWallet = "0123456789abcdef0123456789abcdef01234567"

type transportSource struct {
	work stratum.Work
}

func (s *transportSource) CurrentWork(context.Context, string) (stratum.Work, error) {
	return s.work, nil
}

func (*transportSource) Submit(context.Context, stratum.Candidate) (stratum.SourceResult, error) {
	return stratum.SourceAccepted, nil
}

type transportVerifier struct{}

func (transportVerifier) MeetsTarget(context.Context, stratum.Work, uint64, [32]byte) (bool, error) {
	return false, nil
}

type transportLane uint32

func (l transportLane) Acquire(string, string) (uint32, error) { return uint32(l), nil }
func (transportLane) Release(string, string)                   {}

func newTransportFactory(t *testing.T, calls *atomic.Int32) SessionFactory {
	t.Helper()
	source := &transportSource{work: stratum.Work{
		WorkID:          "transport-work-1",
		Algorithm:       "sudharma-gpupow-v1",
		TargetHex:       "0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f",
		HeaderPrefixHex: "aabbccdd",
		RewardAddress:   transportWallet,
		Version:         2,
		Height:          7500,
		Difficulty:      11,
	}}
	return func() (*stratum.Session, error) {
		calls.Add(1)
		return stratum.NewSession(
			bytes.NewReader(bytes.Repeat([]byte{0x22}, 16)),
			source,
			transportVerifier{},
			stratum.Config{ShareDifficulty: 4, LaneSource: transportLane(0x01020304)},
		)
	}
}

func TestServeConnFragmentedLifecycleAndImmediateWork(t *testing.T) {
	server, client := net.Pipe()
	var calls atomic.Int32
	factory := newTransportFactory(t, &calls)
	done := make(chan error, 1)
	go func() {
		done <- ServeConn(context.Background(), server, factory, Config{})
	}()

	if _, err := io.WriteString(client, `{"id":1,"method":"mining.sub`); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(client, `scribe","params":[]}`+"\n"); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(client)
	subscribe := readTransportJSONLine(t, reader)
	if id := int(subscribe["id"].(float64)); id != 1 {
		t.Fatalf("subscribe response id = %d, want 1", id)
	}
	if subscribe["result"] != "22222222222222222222222222222222" {
		t.Fatalf("subscribe result = %v", subscribe["result"])
	}

	authorize := `{"id":2,"method":"mining.authorize","params":["` + transportWallet + `.rig_01","x"]}` + "\r\n"
	if _, err := io.WriteString(client, authorize); err != nil {
		t.Fatal(err)
	}
	response := readTransportJSONLine(t, reader)
	if response["result"] != true {
		t.Fatalf("authorize result = %v, want true", response["result"])
	}
	difficulty := readTransportJSONLine(t, reader)
	if difficulty["method"] != "mining.set_difficulty" {
		t.Fatalf("first work message method = %v", difficulty["method"])
	}
	notify := readTransportJSONLine(t, reader)
	if notify["method"] != "mining.notify" {
		t.Fatalf("second work message method = %v", notify["method"])
	}

	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ServeConn error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeConn did not return after client EOF")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("session factory calls = %d, want 1", got)
	}
}

func TestServeConnCoalescedRequests(t *testing.T) {
	server, client := net.Pipe()
	var calls atomic.Int32
	factory := newTransportFactory(t, &calls)
	done := make(chan error, 1)
	go func() {
		done <- ServeConn(context.Background(), server, factory, Config{})
	}()

	requests := `{"id":1,"method":"mining.subscribe","params":[]}` + "\n" +
		`{"id":2,"method":"mining.authorize","params":["` + transportWallet + `.rig_01","x"]}` + "\n"
	writeDone := make(chan error, 1)
	go func() {
		_, err := io.WriteString(client, requests)
		writeDone <- err
	}()

	reader := bufio.NewReader(client)
	for i := 0; i < 4; i++ {
		_ = readTransportJSONLine(t, reader)
	}
	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("coalesced client write did not finish")
	}

	_ = client.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ServeConn error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeConn did not return")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("session factory calls = %d, want 1", got)
	}
}

func TestServeConnCRLFSubscribe(t *testing.T) {
	server, client := net.Pipe()
	var calls atomic.Int32
	factory := newTransportFactory(t, &calls)
	done := make(chan error, 1)
	go func() {
		done <- ServeConn(context.Background(), server, factory, Config{})
	}()

	if _, err := io.WriteString(client, `{"id":9,"method":"mining.subscribe","params":[]}`+"\r\n"); err != nil {
		t.Fatal(err)
	}
	got := readTransportJSONLine(t, bufio.NewReader(client))
	if id := int(got["id"].(float64)); id != 9 {
		t.Fatalf("response id = %d, want 9", id)
	}
	_ = client.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ServeConn error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeConn did not return")
	}
}

func readTransportJSONLine(t *testing.T, reader *bufio.Reader) map[string]any {
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
