package demandminer

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type statusResult struct {
	status Status
	err    error
}

type fakeStatusSource struct {
	mu      sync.Mutex
	results []statusResult
	calls   int
}

func (f *fakeStatusSource) Status(context.Context) (Status, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if len(f.results) == 0 {
		return Status{}, errors.New("no status result")
	}
	r := f.results[0]
	if len(f.results) > 1 {
		f.results = f.results[1:]
	}
	return r.status, r.err
}

type fakeMiner struct {
	mu        sync.Mutex
	calls     int
	active    int
	maxActive int
	err       error
}

func (f *fakeMiner) MineOne(context.Context) error {
	f.mu.Lock()
	f.calls++
	f.active++
	if f.active > f.maxActive {
		f.maxActive = f.active
	}
	f.active--
	f.mu.Unlock()
	return f.err
}

type stopSleeper struct {
	mu        sync.Mutex
	durations []time.Duration
	stopAfter int
}

func (s *stopSleeper) Sleep(ctx context.Context, d time.Duration) error {
	s.mu.Lock()
	s.durations = append(s.durations, d)
	shouldStop := s.stopAfter > 0 && len(s.durations) >= s.stopAfter
	s.mu.Unlock()
	if shouldStop {
		return context.Canceled
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

type fakeLogger struct{ errors int }

func (l *fakeLogger) Error(string, error) { l.errors++ }

func validStatus(mempool int) Status {
	return Status{Network: "sudharma", Coin: "Sudharma", Symbol: "SUDH", Height: 10, IssuedSupply: 100, Mempool: mempool}
}

func TestSupervisorEmptyMempoolNeverMines(t *testing.T) {
	source := &fakeStatusSource{results: []statusResult{{status: validStatus(0)}}}
	miner := &fakeMiner{}
	sleeper := &stopSleeper{stopAfter: 1}
	s := NewSupervisor(validConfig(), source, miner, sleeper, &fakeLogger{})
	if err := s.Run(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run: %v", err)
	}
	if miner.calls != 0 {
		t.Fatalf("MineOne calls = %d", miner.calls)
	}
	if got := sleeper.durations; len(got) != 1 || got[0] != 10*time.Second {
		t.Fatalf("sleep durations = %v", got)
	}
}

func TestSupervisorPositiveMempoolMinesOnceThenCooldown(t *testing.T) {
	source := &fakeStatusSource{results: []statusResult{{status: validStatus(2)}}}
	miner := &fakeMiner{}
	sleeper := &stopSleeper{stopAfter: 1}
	s := NewSupervisor(validConfig(), source, miner, sleeper, &fakeLogger{})
	if err := s.Run(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run: %v", err)
	}
	if miner.calls != 1 {
		t.Fatalf("MineOne calls = %d", miner.calls)
	}
	if got := sleeper.durations; len(got) != 1 || got[0] != 30*time.Second {
		t.Fatalf("sleep durations = %v", got)
	}
}

func TestSupervisorRejectsWrongIdentity(t *testing.T) {
	status := validStatus(1)
	status.Network = "other"
	s := NewSupervisor(validConfig(), &fakeStatusSource{results: []statusResult{{status: status}}}, &fakeMiner{}, &stopSleeper{}, &fakeLogger{})
	err := s.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("expected identity error, got %v", err)
	}
}

func TestSupervisorStatusErrorUsesFailureBackoff(t *testing.T) {
	logger := &fakeLogger{}
	sleeper := &stopSleeper{stopAfter: 1}
	s := NewSupervisor(validConfig(), &fakeStatusSource{results: []statusResult{{err: errors.New("rpc down")}}}, &fakeMiner{}, sleeper, logger)
	if err := s.Run(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run: %v", err)
	}
	if logger.errors != 1 {
		t.Fatalf("logger errors = %d", logger.errors)
	}
	if got := sleeper.durations; len(got) != 1 || got[0] != 30*time.Second {
		t.Fatalf("sleep durations = %v", got)
	}
}

func TestSupervisorMinerErrorUsesFailureBackoff(t *testing.T) {
	logger := &fakeLogger{}
	miner := &fakeMiner{err: errors.New("mine failed")}
	sleeper := &stopSleeper{stopAfter: 1}
	s := NewSupervisor(validConfig(), &fakeStatusSource{results: []statusResult{{status: validStatus(1)}}}, miner, sleeper, logger)
	if err := s.Run(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run: %v", err)
	}
	if miner.calls != 1 || logger.errors != 1 {
		t.Fatalf("calls=%d errors=%d", miner.calls, logger.errors)
	}
	if got := sleeper.durations; len(got) != 1 || got[0] != 30*time.Second {
		t.Fatalf("sleep durations = %v", got)
	}
}

func TestSupervisorRemainingMempoolWaitsCooldownBeforeNextBlock(t *testing.T) {
	source := &fakeStatusSource{results: []statusResult{{status: validStatus(2)}, {status: validStatus(1)}}}
	miner := &fakeMiner{}
	sleeper := &stopSleeper{stopAfter: 2}
	s := NewSupervisor(validConfig(), source, miner, sleeper, &fakeLogger{})
	if err := s.Run(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run: %v", err)
	}
	if miner.calls != 2 {
		t.Fatalf("MineOne calls = %d", miner.calls)
	}
	if got := sleeper.durations; len(got) != 2 || got[0] != 30*time.Second || got[1] != 30*time.Second {
		t.Fatalf("sleep durations = %v", got)
	}
	if miner.maxActive != 1 {
		t.Fatalf("max active miners = %d", miner.maxActive)
	}
}
