package stratum

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"sync"
	"testing"
)

type submitSource struct {
	mu         sync.Mutex
	work       Work
	result     SourceResult
	err        error
	candidates []Candidate
}

func (s *submitSource) CurrentWork(context.Context, string) (Work, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.work, nil
}
func (s *submitSource) Submit(_ context.Context, c Candidate) (SourceResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.candidates = append(s.candidates, c)
	return s.result, s.err
}
func (s *submitSource) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.candidates)
}
func (s *submitSource) last() Candidate {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.candidates[len(s.candidates)-1]
}
func (s *submitSource) setWork(work Work) {
	s.mu.Lock()
	s.work = work
	s.mu.Unlock()
}

type verifyReply struct {
	ok  bool
	err error
}
type verifyCall struct {
	work   Work
	nonce  uint64
	target [32]byte
}
type submitVerifier struct {
	mu      sync.Mutex
	replies []verifyReply
	calls   []verifyCall
}

func (v *submitVerifier) MeetsTarget(_ context.Context, work Work, nonce uint64, target [32]byte) (bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.calls = append(v.calls, verifyCall{work: work, nonce: nonce, target: target})
	if len(v.replies) == 0 {
		return false, errors.New("unexpected verifier call")
	}
	r := v.replies[0]
	v.replies = v.replies[1:]
	return r.ok, r.err
}
func (v *submitVerifier) callCount() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.calls)
}
func (v *submitVerifier) call(i int) verifyCall {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.calls[i]
}

type fixedSubmitLane struct{ lane uint32 }

func (f fixedSubmitLane) Acquire(string, string) (uint32, error) { return f.lane, nil }
func (fixedSubmitLane) Release(string, string)                   {}

func TestSessionSubmitHandleRoutesRequest(t *testing.T) {
	s, _, verifier := newSubmitFixture(4)
	verifier.replies = []verifyReply{{ok: false}}
	line := mustSubmitJSON(map[string]any{
		"id": 1, "method": "mining.submit",
		"params": []string{submitUser(), "job-1", strconv.FormatUint(laneNonce(7, 1), 16)},
	})
	messages, err := s.Handle(context.Background(), line)
	if err != nil || submitStatus(t, messages) != SubmitInvalid {
		t.Fatalf("handle submit=%v err=%v", messages, err)
	}
}

func TestSessionSubmitValidation(t *testing.T) {
	s, source, verifier := newSubmitFixture(4)
	s.identity = nil
	_, err := s.handleSubmit(context.Background(), submitReq(1, submitUser(), s.currentJob.id, laneNonce(7, 1)))
	assertSubmitCode(t, err, protocolInvalidRequest)
	s, source, verifier = newSubmitFixture(4)

	cases := []struct {
		name   string
		worker string
		jobID  string
		nonce  uint64
		want   SubmitStatus
	}{
		{"worker", "9ccdc094489874bed888ffe4bdf9b8298f4c5131.other", "job-1", laneNonce(7, 1), SubmitInvalid},
		{"job", submitUser(), "unknown", laneNonce(7, 1), SubmitInvalid},
		{"lane", submitUser(), "job-1", laneNonce(8, 1), SubmitInvalid},
		{"stale", submitUser(), "old-job", laneNonce(7, 1), SubmitStale},
	}
	s.staleJobIDs = []string{"old-job"}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.handleSubmit(context.Background(), submitReq(1, tc.worker, tc.jobID, tc.nonce))
			if err != nil {
				t.Fatal(err)
			}
			if status := submitStatus(t, got); status != tc.want {
				t.Fatalf("status=%q want %q", status, tc.want)
			}
		})
	}
	if source.count() != 0 || verifier.callCount() != 0 {
		t.Fatal("invalid submissions reached verifier/source")
	}

	for _, nonce := range []string{"", "-1", "0x1", "10000000000000000", "xyz", " 01"} {
		t.Run("nonce="+nonce, func(t *testing.T) {
			req := Request{ID: json.RawMessage(`1`), Method: "mining.submit", Params: mustSubmitJSON([]string{submitUser(), "job-1", nonce})}
			_, err := s.handleSubmit(context.Background(), req)
			assertSubmitCode(t, err, protocolInvalidParams)
		})
	}
}

func TestSessionSubmitShareAndCandidateClassification(t *testing.T) {
	t.Run("invalid share", func(t *testing.T) {
		s, source, verifier := newSubmitFixture(4)
		verifier.replies = []verifyReply{{ok: false}}
		got, err := s.handleSubmit(context.Background(), submitReq(1, submitUser(), "job-1", laneNonce(7, 2)))
		if err != nil || submitStatus(t, got) != SubmitInvalid || source.count() != 0 {
			t.Fatalf("got=%v err=%v source=%d", got, err, source.count())
		}
	})

	t.Run("accepted share", func(t *testing.T) {
		s, source, verifier := newSubmitFixture(4)
		verifier.replies = []verifyReply{{ok: true}, {ok: false}}
		nonce := laneNonce(7, 3)
		got, err := s.handleSubmit(context.Background(), submitReq(1, submitUser(), "job-1", nonce))
		if err != nil || submitStatus(t, got) != SubmitAcceptedShare || source.count() != 0 {
			t.Fatalf("got=%v err=%v source=%d", got, err, source.count())
		}
		if verifier.call(0).target != submitTarget(4) || verifier.call(1).target != decodeSubmitTarget(s.currentJob.work.TargetHex) {
			t.Fatal("share/network targets do not match immutable work contract")
		}
	})

	mappings := []struct {
		source SourceResult
		want   SubmitStatus
	}{
		{SourceAccepted, SubmitAcceptedBlock},
		{SourceInvalid, SubmitInvalid},
		{SourceStale, SubmitStale},
		{SourceMutated, SubmitMutated},
	}
	for _, m := range mappings {
		t.Run(string(m.source), func(t *testing.T) {
			s, source, verifier := newSubmitFixture(4)
			source.result = m.source
			verifier.replies = []verifyReply{{ok: true}, {ok: true}}
			nonce := laneNonce(7, 4)
			got, err := s.handleSubmit(context.Background(), submitReq(1, submitUser(), "job-1", nonce))
			if err != nil || submitStatus(t, got) != m.want || source.count() != 1 {
				t.Fatalf("status=%v err=%v source=%d", got, err, source.count())
			}
			c := source.last()
			if c.Work != s.currentJob.work || c.JobID != "job-1" || c.Identity != *s.identity || c.Lane != 7 || c.Nonce != nonce {
				t.Fatalf("candidate mutated: %+v", c)
			}
		})
	}
}

func TestSessionSubmitDuplicateAndBounds(t *testing.T) {
	s, source, verifier := newSubmitFixture(2)
	verifier.replies = []verifyReply{{ok: false}, {ok: false}}
	first := submitReq(1, submitUser(), "job-1", laneNonce(7, 1))
	got, err := s.handleSubmit(context.Background(), first)
	if err != nil || submitStatus(t, got) != SubmitInvalid {
		t.Fatalf("first submit: %v %v", got, err)
	}
	got, err = s.handleSubmit(context.Background(), first)
	if err != nil || submitStatus(t, got) != SubmitDuplicate || source.count() != 0 {
		t.Fatalf("duplicate: %v %v", got, err)
	}
	if _, err := s.handleSubmit(context.Background(), submitReq(2, submitUser(), "job-1", laneNonce(7, 2))); err != nil {
		t.Fatal(err)
	}
	if _, err := s.handleSubmit(context.Background(), submitReq(3, submitUser(), "job-1", laneNonce(7, 3))); err == nil {
		t.Fatal("expected configured duplicate tracker limit to fail closed")
	}

	s, _, _ = newSubmitFixture(defaultMaxDuplicateShares)
	s.duplicateShares = make(map[shareKey]struct{}, defaultMaxDuplicateShares)
	for i := 0; i < defaultMaxDuplicateShares; i++ {
		s.duplicateShares[shareKey{jobID: "job-1", nonce: uint64(i)}] = struct{}{}
	}
	if _, err := s.handleSubmit(context.Background(), submitReq(4, submitUser(), "job-1", laneNonce(7, uint64(defaultMaxDuplicateShares+1)))); err == nil {
		t.Fatal("expected 65,536-entry duplicate tracker to fail closed")
	}
}

func TestSessionSubmitOperationalErrors(t *testing.T) {
	s, _, verifier := newSubmitFixture(4)
	verifier.replies = []verifyReply{{err: errors.New("verifier unavailable")}, {ok: false}}
	req := submitReq(1, submitUser(), "job-1", laneNonce(7, 5))
	if _, err := s.handleSubmit(context.Background(), req); err == nil {
		t.Fatal("expected verifier error")
	}
	got, err := s.handleSubmit(context.Background(), req)
	if err != nil || submitStatus(t, got) != SubmitInvalid {
		t.Fatalf("verifier retry=%v err=%v", got, err)
	}

	s, source, verifier := newSubmitFixture(4)
	source.err = errors.New("submit uncertain")
	verifier.replies = []verifyReply{{ok: true}, {ok: true}}
	req = submitReq(1, submitUser(), "job-1", laneNonce(7, 6))
	if _, err := s.handleSubmit(context.Background(), req); err == nil {
		t.Fatal("expected source error")
	}
	got, err = s.handleSubmit(context.Background(), req)
	if err != nil || submitStatus(t, got) != SubmitDuplicate || source.count() != 1 {
		t.Fatalf("uncertain forward retry=%v err=%v source=%d", got, err, source.count())
	}
}

func TestSessionSubmitCleanWorkClearsDuplicateTracker(t *testing.T) {
	s, source, verifier := newSubmitFixture(1)
	verifier.replies = []verifyReply{{ok: false}, {ok: false}}
	if _, err := s.handleSubmit(context.Background(), submitReq(1, submitUser(), "job-1", laneNonce(7, 1))); err != nil {
		t.Fatal(err)
	}
	source.setWork(submitWork("work-2", s.identity.Wallet))
	s.config.LaneSource = fixedSubmitLane{lane: 8}
	if _, err := s.RefreshWork(context.Background()); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	jobID, lane := s.currentJob.id, s.currentJob.lane
	s.mu.Unlock()
	got, err := s.handleSubmit(context.Background(), submitReq(2, submitUser(), jobID, laneNonce(lane, 1)))
	if err != nil || submitStatus(t, got) != SubmitInvalid {
		t.Fatalf("clean work did not reset duplicate capacity: %v %v", got, err)
	}
}

func TestSessionSubmitConcurrentSameNonceHasOneWinner(t *testing.T) {
	s, source, verifier := newSubmitFixture(4)
	verifier.replies = []verifyReply{{ok: true}, {ok: true}}
	req := submitReq(1, submitUser(), "job-1", laneNonce(7, 7))
	const workers = 16
	statuses := make(chan SubmitStatus, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			messages, err := s.handleSubmit(context.Background(), req)
			if err != nil {
				errs <- err
				return
			}
			status, err := submitStatusValue(messages)
			if err != nil {
				errs <- err
				return
			}
			statuses <- status
		}()
	}
	wg.Wait()
	close(statuses)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	winner, duplicates := 0, 0
	for status := range statuses {
		switch status {
		case SubmitAcceptedBlock:
			winner++
		case SubmitDuplicate:
			duplicates++
		default:
			t.Fatalf("unexpected status %q", status)
		}
	}
	if winner != 1 || duplicates != workers-1 || source.count() != 1 {
		t.Fatalf("winner=%d duplicates=%d source=%d", winner, duplicates, source.count())
	}
}

func newSubmitFixture(limit int) (*Session, *submitSource, *submitVerifier) {
	identity := WorkerIdentity{Wallet: "9ccdc094489874bed888ffe4bdf9b8298f4c5131", Worker: "rig_01"}
	work := submitWork("work-1", identity.Wallet)
	source := &submitSource{work: work, result: SourceAccepted}
	verifier := &submitVerifier{}
	s := &Session{
		id: "session", source: source, verifier: verifier,
		config:     Config{ShareDifficulty: 4, MaxDuplicateShares: limit},
		subscribed: true, identity: &identity, generation: 1,
		currentJob: &job{id: "job-1", work: work, generation: 1, lane: 7},
	}
	return s, source, verifier
}

func submitWork(id, reward string) Work {
	target := submitTarget(11)
	return Work{WorkID: id, Algorithm: "khushi-gpu-v1", TargetHex: hex.EncodeToString(target[:]), HeaderPrefixHex: "aabb", RewardAddress: reward, Version: 2, Height: 100, Difficulty: 11}
}
func submitUser() string                       { return "9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01" }
func laneNonce(lane uint32, low uint64) uint64 { return uint64(lane)<<32 | low&0xffffffff }
func submitReq(id int, worker, jobID string, nonce uint64) Request {
	return Request{ID: json.RawMessage(strconv.Itoa(id)), Method: "mining.submit", Params: mustSubmitJSON([]string{worker, jobID, strconv.FormatUint(nonce, 16)})}
}
func mustSubmitJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}
func submitStatus(t *testing.T, messages []Message) SubmitStatus {
	t.Helper()
	status, err := submitStatusValue(messages)
	if err != nil {
		t.Fatal(err)
	}
	return status
}
func submitStatusValue(messages []Message) (SubmitStatus, error) {
	if len(messages) != 1 {
		return "", errors.New("expected one submit response")
	}
	response, ok := messages[0].(Response)
	if !ok {
		return "", errors.New("unexpected submit response type")
	}
	status, ok := response.Result.(SubmitStatus)
	if !ok {
		return "", errors.New("unexpected submit result type")
	}
	return status, nil
}
func assertSubmitCode(t *testing.T, err error, code int) {
	t.Helper()
	var protocolErr *ProtocolError
	if !errors.As(err, &protocolErr) || protocolErr.Code != code {
		t.Fatalf("error=%v want protocol code %d", err, code)
	}
}
func submitTarget(difficulty uint32) [32]byte {
	maxHash := new(big.Int).Lsh(big.NewInt(1), 256)
	maxHash.Sub(maxHash, big.NewInt(1))
	maxHash.Div(maxHash, new(big.Int).SetUint64(uint64(difficulty)))
	var target [32]byte
	maxHash.FillBytes(target[:])
	return target
}
func decodeSubmitTarget(value string) [32]byte {
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		panic("invalid test target")
	}
	var target [32]byte
	copy(target[:], decoded)
	return target
}
