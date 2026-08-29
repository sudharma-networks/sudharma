package stratum

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"testing"
)

type refreshTestSource struct {
	work Work
	err  error
}

func (s *refreshTestSource) CurrentWork(context.Context, string) (Work, error) {
	return s.work, s.err
}
func (s *refreshTestSource) Submit(context.Context, Candidate) (SourceResult, error) {
	return SourceAccepted, nil
}

func TestSessionRefreshWorkEmitsDifficultyThenNotify(t *testing.T) {
	source := &refreshTestSource{work: refreshWork("work-1")}
	allocator := NewLaneAllocator()
	s := newRefreshSession(t, source, allocator, 17)
	authorizeRefreshSession(t, s)

	messages, err := s.RefreshWork(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 {
		t.Fatalf("message count = %d, want 2", len(messages))
	}

	difficulty, ok := messages[0].(Notification)
	if !ok || difficulty.Method != "mining.set_difficulty" {
		t.Fatalf("first message = %#v, want mining.set_difficulty", messages[0])
	}
	difficultyParams, ok := difficulty.Params.([]any)
	if !ok || len(difficultyParams) != 1 || difficultyParams[0] != uint32(17) {
		t.Fatalf("difficulty params = %#v", difficulty.Params)
	}

	notify, ok := messages[1].(Notification)
	if !ok || notify.Method != "mining.notify" {
		t.Fatalf("second message = %#v, want mining.notify", messages[1])
	}
	params, ok := notify.Params.(notifyParams)
	if !ok {
		t.Fatalf("notify params type = %T, want notifyParams", notify.Params)
	}
	wantJobID := testJobID(source.work.WorkID, s.SessionID(), 1)
	wantLane := testLaneStart(source.work.WorkID, s.SessionID())
	if params.JobID != wantJobID || params.Algorithm != source.work.Algorithm || params.Height != source.work.Height ||
		params.TargetHex != source.work.TargetHex || params.HeaderPrefixHex != source.work.HeaderPrefixHex ||
		params.RewardAddress != source.work.RewardAddress || params.Version != source.work.Version ||
		params.NetworkDifficulty != source.work.Difficulty || params.Lane != wantLane || !params.CleanJobs {
		t.Fatalf("unexpected notify params: %+v", params)
	}
}

func TestSessionRefreshWorkIdenticalWorkEmitsNothing(t *testing.T) {
	source := &refreshTestSource{work: refreshWork("work-1")}
	s := newRefreshSession(t, source, NewLaneAllocator(), 1)
	authorizeRefreshSession(t, s)
	if _, err := s.RefreshWork(context.Background()); err != nil {
		t.Fatal(err)
	}

	messages, err := s.RefreshWork(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Fatalf("identical work emitted %d messages", len(messages))
	}
}

func TestSessionRefreshWorkChangedIDIncrementsGenerationAndStalesOldJob(t *testing.T) {
	source := &refreshTestSource{work: refreshWork("work-1")}
	allocator := NewLaneAllocator()
	s := newRefreshSession(t, source, allocator, 3)
	authorizeRefreshSession(t, s)
	first, err := s.RefreshWork(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	firstParams := first[1].(Notification).Params.(notifyParams)
	oldLane := firstParams.Lane

	source.work = refreshWork("work-2")
	second, err := s.RefreshWork(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	secondParams := second[1].(Notification).Params.(notifyParams)
	if secondParams.JobID != testJobID("work-2", s.SessionID(), 2) {
		t.Fatalf("new job ID = %q", secondParams.JobID)
	}
	if !sessionHasStaleJob(s, firstParams.JobID) {
		t.Fatalf("old job %q was not retained as stale", firstParams.JobID)
	}
	if laneAssignmentExists(allocator, "work-1", oldLane) {
		t.Fatalf("old lane %d was not released", oldLane)
	}
}

func TestSessionRefreshWorkRejectsMutatedReuseOfWorkID(t *testing.T) {
	source := &refreshTestSource{work: refreshWork("work-1")}
	s := newRefreshSession(t, source, NewLaneAllocator(), 1)
	authorizeRefreshSession(t, s)
	first, err := s.RefreshWork(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	firstParams := first[1].(Notification).Params.(notifyParams)

	mutated := source.work
	mutated.TargetHex = "ff00"
	source.work = mutated
	messages, err := s.RefreshWork(context.Background())
	if err == nil {
		t.Fatal("expected mutated work-ID reuse to fail")
	}
	if len(messages) != 0 {
		t.Fatalf("mutated work emitted %d messages", len(messages))
	}
	if got := sessionCurrentJobID(s); got != firstParams.JobID {
		t.Fatalf("current job changed to %q after mutation failure", got)
	}
}

func TestSessionRefreshWorkRequiresAuthorization(t *testing.T) {
	source := &refreshTestSource{work: refreshWork("work-1")}
	s := newRefreshSession(t, source, NewLaneAllocator(), 1)
	if _, err := s.RefreshWork(context.Background()); err == nil {
		t.Fatal("expected unauthorized refresh to fail")
	}
}

func TestSessionRefreshWorkLaneCollisionProbesUpward(t *testing.T) {
	allocator := NewLaneAllocator()
	workID := "work-1"
	sessionID := hex.EncodeToString([]byte("0123456789abcdef"))
	start := testLaneStart(workID, sessionID)
	reserveLaneForTest(allocator, workID, start, "other-session")

	lane, err := allocator.Acquire(workID, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if lane != start+1 {
		t.Fatalf("lane = %d, want %d", lane, start+1)
	}
}

func TestSessionRefreshWorkRetainsAtMostEightStaleIDs(t *testing.T) {
	source := &refreshTestSource{work: refreshWork("work-0")}
	s := newRefreshSession(t, source, NewLaneAllocator(), 1)
	authorizeRefreshSession(t, s)
	for i := 0; i < 10; i++ {
		source.work = refreshWork("work-" + string(rune('a'+i)))
		if _, err := s.RefreshWork(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if got := sessionStaleJobCount(s); got > 8 {
		t.Fatalf("stale job count = %d, want <= 8", got)
	}
}

func TestSessionRefreshWorkPropagatesSourceErrorWithoutStateChange(t *testing.T) {
	source := &refreshTestSource{work: refreshWork("work-1")}
	s := newRefreshSession(t, source, NewLaneAllocator(), 1)
	authorizeRefreshSession(t, s)
	source.err = errors.New("source unavailable")
	if _, err := s.RefreshWork(context.Background()); err == nil {
		t.Fatal("expected source error")
	}
	if got := sessionCurrentJobID(s); got != "" {
		t.Fatalf("current job = %q after source error", got)
	}
}

func refreshWork(id string) Work {
	return Work{
		WorkID:          id,
		Algorithm:       "khushi-gpu-v1",
		TargetHex:       "0000ffff",
		HeaderPrefixHex: "aabbccdd",
		RewardAddress:   "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
		Version:         2,
		Height:          100,
		Difficulty:      11,
	}
}

func newRefreshSession(t *testing.T, source WorkSource, lanes LaneSource, shareDifficulty uint32) *Session {
	t.Helper()
	s, err := NewSession(bytes.NewReader([]byte("0123456789abcdef")), source, sessionTestVerifier{}, Config{
		ShareDifficulty: shareDifficulty,
		LaneSource:      lanes,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func authorizeRefreshSession(t *testing.T, s *Session) {
	t.Helper()
	subscribeSession(t, s)
	if _, err := s.Handle(context.Background(), authorizeRequest(2, "9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01", "x")); err != nil {
		t.Fatal(err)
	}
}

func testJobID(workID, sessionID string, generation uint64) string {
	h := sha256.New()
	h.Write([]byte("SUDHARMA-STRATUM-JOB-V1\x00"))
	h.Write([]byte(workID))
	h.Write([]byte(sessionID))
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], generation)
	h.Write(buf[:])
	return hex.EncodeToString(h.Sum(nil))
}

func testLaneStart(workID, sessionID string) uint32 {
	h := sha256.New()
	h.Write([]byte("SUDHARMA-STRATUM-LANE-V1\x00"))
	h.Write([]byte(workID))
	h.Write([]byte(sessionID))
	sum := h.Sum(nil)
	return binary.BigEndian.Uint32(sum[:4])
}

func sessionHasStaleJob(s *Session, jobID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, staleID := range s.staleJobIDs {
		if staleID == jobID {
			return true
		}
	}
	return false
}

func sessionStaleJobCount(s *Session) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.staleJobIDs)
}

func sessionCurrentJobID(s *Session) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentJob == nil {
		return ""
	}
	return s.currentJob.id
}

func reserveLaneForTest(a *LaneAllocator, workID string, lane uint32, sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.byWork[workID] == nil {
		a.byWork[workID] = make(map[uint32]string)
	}
	a.byWork[workID][lane] = sessionID
	a.bySession[laneSessionKey{workID: workID, sessionID: sessionID}] = lane
}

func laneAssignmentExists(a *LaneAllocator, workID string, lane uint32) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.byWork[workID][lane]
	return ok
}
