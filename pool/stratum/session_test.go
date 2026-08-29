package stratum

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"testing"
)

type sessionTestSource struct{}

func (sessionTestSource) CurrentWork(context.Context, string) (Work, error) { return Work{}, nil }
func (sessionTestSource) Submit(context.Context, Candidate) (SourceResult, error) {
	return SourceAccepted, nil
}

type sessionTestVerifier struct{}

func (sessionTestVerifier) MeetsTarget(context.Context, Work, uint64, [32]byte) (bool, error) {
	return false, nil
}

type sessionTestLanes struct{}

func (sessionTestLanes) Acquire(string, string) (uint32, error) { return 1, nil }
func (sessionTestLanes) Release(string, string)                {}

func TestNewSessionUsesExactly128BitsOfEntropy(t *testing.T) {
	entropy := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	s, err := NewSession(bytes.NewReader(entropy), sessionTestSource{}, sessionTestVerifier{}, Config{
		ShareDifficulty: 1,
		LaneSource:      sessionTestLanes{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := s.SessionID(), hex.EncodeToString(entropy); got != want {
		t.Fatalf("session ID = %q, want %q", got, want)
	}
}

func TestNewSessionRejectsEntropyShortRead(t *testing.T) {
	_, err := NewSession(bytes.NewReader(make([]byte, 15)), sessionTestSource{}, sessionTestVerifier{}, Config{
		ShareDifficulty: 1,
		LaneSource:      sessionTestLanes{},
	})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("error = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestSessionSubscribeReturnsStableSessionID(t *testing.T) {
	s := newSessionForLifecycleTest(t)
	messages, err := s.Handle(context.Background(), []byte(`{"id":1,"method":"mining.subscribe","params":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	response := onlyResponse(t, messages)
	if got, ok := response.Result.(string); !ok || got != s.SessionID() {
		t.Fatalf("subscribe result = %#v, want session ID %q", response.Result, s.SessionID())
	}
	if string(response.ID) != "1" || response.Error != nil {
		t.Fatalf("unexpected subscribe response: %+v", response)
	}
}

func TestSessionAuthorizeRequiresSubscribe(t *testing.T) {
	s := newSessionForLifecycleTest(t)
	_, err := s.Handle(context.Background(), authorizeRequest(1, "9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01", "x"))
	assertProtocolErrorCode(t, err, -32600)
}

func TestSessionAuthorizeAcceptsCompatibilityPasswords(t *testing.T) {
	for _, password := range []string{"", "x"} {
		t.Run("password="+password, func(t *testing.T) {
			s := newSessionForLifecycleTest(t)
			subscribeSession(t, s)
			messages, err := s.Handle(context.Background(), authorizeRequest(2, "9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01", password))
			if err != nil {
				t.Fatal(err)
			}
			response := onlyResponse(t, messages)
			if got, ok := response.Result.(bool); !ok || !got {
				t.Fatalf("authorize result = %#v, want true", response.Result)
			}
		})
	}
}

func TestSessionAuthorizeRejectsOtherPassword(t *testing.T) {
	s := newSessionForLifecycleTest(t)
	subscribeSession(t, s)
	_, err := s.Handle(context.Background(), authorizeRequest(2, "9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01", "secret"))
	assertProtocolErrorCode(t, err, -32602)
}

func TestSessionAuthorizeIsIdempotentForSameIdentity(t *testing.T) {
	s := newSessionForLifecycleTest(t)
	subscribeSession(t, s)
	worker := "9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01"
	for id := 2; id <= 3; id++ {
		messages, err := s.Handle(context.Background(), authorizeRequest(id, worker, "x"))
		if err != nil {
			t.Fatal(err)
		}
		response := onlyResponse(t, messages)
		if response.Result != true {
			t.Fatalf("authorize result = %#v, want true", response.Result)
		}
	}
}

func TestSessionAuthorizeRejectsDifferentReauthorization(t *testing.T) {
	s := newSessionForLifecycleTest(t)
	subscribeSession(t, s)
	if _, err := s.Handle(context.Background(), authorizeRequest(2, "9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01", "x")); err != nil {
		t.Fatal(err)
	}
	_, err := s.Handle(context.Background(), authorizeRequest(3, "9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_02", "x"))
	assertProtocolErrorCode(t, err, -32600)
}

func newSessionForLifecycleTest(t *testing.T) *Session {
	t.Helper()
	s, err := NewSession(bytes.NewReader([]byte("0123456789abcdef")), sessionTestSource{}, sessionTestVerifier{}, Config{
		ShareDifficulty: 1,
		LaneSource:      sessionTestLanes{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func subscribeSession(t *testing.T, s *Session) {
	t.Helper()
	if _, err := s.Handle(context.Background(), []byte(`{"id":1,"method":"mining.subscribe","params":[]}`)); err != nil {
		t.Fatal(err)
	}
}

func authorizeRequest(id int, worker, password string) []byte {
	data, _ := json.Marshal(map[string]any{
		"id":     id,
		"method": "mining.authorize",
		"params": []string{worker, password},
	})
	return data
}

func onlyResponse(t *testing.T, messages []Message) Response {
	t.Helper()
	if len(messages) != 1 {
		t.Fatalf("message count = %d, want 1", len(messages))
	}
	response, ok := messages[0].(Response)
	if !ok {
		t.Fatalf("message type = %T, want Response", messages[0])
	}
	return response
}

func assertProtocolErrorCode(t *testing.T, err error, code int) {
	t.Helper()
	if err == nil {
		t.Fatal("expected protocol error")
	}
	var protocolErr *ProtocolError
	if !errors.As(err, &protocolErr) {
		t.Fatalf("error type = %T, want *ProtocolError: %v", err, err)
	}
	if protocolErr.Code != code {
		t.Fatalf("protocol code = %d, want %d", protocolErr.Code, code)
	}
}
