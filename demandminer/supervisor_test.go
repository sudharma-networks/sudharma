package demandminer

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type scriptedStatusSource struct {
	mu       sync.Mutex
	results  []statusResult
	calls    int
	callHook func()
}

type statusResult struct {
	status Status
	err    error
}

func (s *scriptedStatusSource) Status(context.Context) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.callHook != nil {
		s.callHook()
	}
	if len(s.results) == 0 {
		return Status{}, errors.New("unexpected status request")
	}
	result := s.results[0]
	s.results = s.results[1:]
	return result.status, result.err
}

func (s *scriptedStatusSource) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

type fakeMiner struct {
	mu          sync.Mutex
	calls       int
	active      int
	maxActive   int
	results     []error
	start       chan struct{}
	block       <-chan struct{}
}

func (m *fakeMiner) MineOne(ctx context.Context) error {
	m.mu.Lock()
	m.calls++
	m.active++
	if m.active > m.maxActive {
		m.maxActive = m.active
	}
	call := m.calls
	m.mu.Unlock()

	if m.start != nil {
		select {
		case m.start <- struct{}{}:
		default:
		}
	}
	if m.block != nil {
		select {
		case <-m.block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.active--
	if call <= len(m.results) {
		return m.results[call-1]
	}
	return nil
}

func (m *fakeMiner) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func (m *fakeMiner) MaxActive() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.maxActive
}

type recordingSleeper struct {
	mu        sync.Mutex
	durations []time.Duration
	after     func(int)
}

func (s *recordingSleeper) Sleep(ctx context.Context, duration time.Duration) error {
	s.mu.Lock()
	s.durations = append(s.durations, duration)
	call := len(s.durations)
	s.mu.Unlock()
	if s.after != nil {
		s.after(call)
	}
	return ctx.Err()
}

func (s *recordingSleeper) Durations() []time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]time.Duration(nil), s.durations...)
}

type blockingSleeper struct {
	entered chan<- time.Duration
	release <-chan struct{}
}

func (s *blockingSleeper) Sleep(ctx context.Context, duration time.Duration) error {
	select {
	case s.entered <- duration:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-s.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type recordingLogger struct {
	mu     sync.Mutex
	events []string
}

func (l *recordingLogger) Error(event string, _ map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, event)
}

func (l *recordingLogger) Events() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.events...)
}

func supervisorConfig() Config {
	return validConfig()
}

func matchingStatus(mempool int) Status {
	return Status{Network: "sudharma", Coin: "Sudharma", Symbol: "SUDH", Height: 7, IssuedSupply: 350, Mempool: mempool}
}

func newTestSupervisor(cfg Config, source StatusSource, miner BlockMiner, sleeper Sleeper, logger Logger) *Supervisor {
	return NewSupervisor(cfg, source, miner, sleeper, logger)
}

func TestSupervisorEmptyMempoolDoesNotMine(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sleeper := &recordingSleeper{after: func(int) { cancel() }}
	miner := &fakeMiner{}
	supervisor := newTestSupervisor(supervisorConfig(), &scriptedStatusSource{results: []statusResult{{status: matchingStatus(0)}}}, miner, sleeper, nil)

	err := supervisor.Run(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if got := miner.Calls(); got != 0 {
		t.Fatalf("MineOne calls = %d, want 0", got)
	}
	if got, want := sleeper.Durations(), []time.Duration{10 * time.Second}; !equalDurations(got, want) {
		t.Fatalf("sleep durations = %v, want %v", got, want)
	}
}

func TestSupervisorPositiveMempoolMinesOneBlockThenCoolsDown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sleeper := &recordingSleeper{after: func(int) { cancel() }}
	miner := &fakeMiner{}
	supervisor := newTestSupervisor(supervisorConfig(), &scriptedStatusSource{results: []statusResult{{status: matchingStatus(1)}}}, miner, sleeper, nil)

	err := supervisor.Run(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if got := miner.Calls(); got != 1 {
		t.Fatalf("MineOne calls = %d, want 1", got)
	}
	if got, want := sleeper.Durations(), []time.Duration{30 * time.Second}; !equalDurations(got, want) {
		t.Fatalf("sleep durations = %v, want %v", got, want)
	}
}

func TestSupervisorWrongIdentityFailsClosed(t *testing.T) {
	for _, test := range []struct {
		name string
		mutate func(*Status)
	}{
		{name: "network", mutate: func(status *Status) { status.Network = "mainnet" }},
		{name: "coin", mutate: func(status *Status) { status.Coin = "Not Sudharma" }},
		{name: "symbol", mutate: func(status *Status) { status.Symbol = "MAIN" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			status := matchingStatus(1)
			test.mutate(&status)
			miner := &fakeMiner{}
			supervisor := newTestSupervisor(supervisorConfig(), &scriptedStatusSource{results: []statusResult{{status: status}}}, miner, &recordingSleeper{}, nil)

			err := supervisor.Run(context.Background())

			if err == nil {
				t.Fatal("Run error = nil, want identity mismatch")
			}
			if got := miner.Calls(); got != 0 {
				t.Fatalf("MineOne calls = %d, want 0", got)
			}
		})
	}
}

func TestSupervisorRejectsUnvalidatedMainnetConfigBeforePolling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := supervisorConfig()
	cfg.ExpectedNetwork = "mainnet"
	status := matchingStatus(1)
	status.Network = "mainnet"
	source := &scriptedStatusSource{results: []statusResult{{status: status}}}
	miner := &fakeMiner{}
	supervisor := newTestSupervisor(cfg, source, miner, &recordingSleeper{after: func(int) { cancel() }}, nil)

	err := supervisor.Run(ctx)

	if err == nil || !strings.Contains(err.Error(), "invalid demand miner config") {
		t.Fatalf("Run error = %v, want rejected configuration", err)
	}
	if got := source.Calls(); got != 0 {
		t.Fatalf("status calls = %d, want 0", got)
	}
	if got := miner.Calls(); got != 0 {
		t.Fatalf("MineOne calls = %d, want 0", got)
	}
}

func TestSupervisorIgnoresMutatedPublicConfigIdentity(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	status := matchingStatus(1)
	status.Network = "mainnet"
	miner := &fakeMiner{}
	supervisor := newTestSupervisor(supervisorConfig(), &scriptedStatusSource{results: []statusResult{{status: status}}}, miner, &recordingSleeper{after: func(int) { cancel() }}, nil)
	supervisor.Config.ExpectedNetwork = "mainnet"

	err := supervisor.Run(ctx)

	if err == nil || !strings.Contains(err.Error(), "status identity mismatch") {
		t.Fatalf("Run error = %v, want fixed identity rejection", err)
	}
	if got := miner.Calls(); got != 0 {
		t.Fatalf("MineOne calls = %d, want 0", got)
	}
}

func TestSupervisorNegativeMempoolUsesFailureBackoffWithoutMining(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sleeper := &recordingSleeper{after: func(int) { cancel() }}
	miner := &fakeMiner{}
	supervisor := newTestSupervisor(supervisorConfig(), &scriptedStatusSource{results: []statusResult{{status: matchingStatus(-1)}}}, miner, sleeper, nil)

	err := supervisor.Run(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if got := miner.Calls(); got != 0 {
		t.Fatalf("MineOne calls = %d, want 0", got)
	}
	if got, want := sleeper.Durations(), []time.Duration{30 * time.Second}; !equalDurations(got, want) {
		t.Fatalf("sleep durations = %v, want %v", got, want)
	}
}

func TestSupervisorStatusErrorUsesFailureBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sleeper := &recordingSleeper{after: func(int) { cancel() }}
	logger := &recordingLogger{}
	supervisor := newTestSupervisor(supervisorConfig(), &scriptedStatusSource{results: []statusResult{{err: errors.New("status unavailable")}}}, &fakeMiner{}, sleeper, logger)

	err := supervisor.Run(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if got, want := sleeper.Durations(), []time.Duration{30 * time.Second}; !equalDurations(got, want) {
		t.Fatalf("sleep durations = %v, want %v", got, want)
	}
	if got, want := logger.Events(), []string{"status_failed"}; !equalStrings(got, want) {
		t.Fatalf("log events = %v, want %v", got, want)
	}
}

func TestSupervisorMinerErrorUsesFailureBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sleeper := &recordingSleeper{after: func(int) { cancel() }}
	logger := &recordingLogger{}
	miner := &fakeMiner{results: []error{errors.New("miner failed")}}
	supervisor := newTestSupervisor(supervisorConfig(), &scriptedStatusSource{results: []statusResult{{status: matchingStatus(1)}}}, miner, sleeper, logger)

	err := supervisor.Run(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if got, want := sleeper.Durations(), []time.Duration{30 * time.Second}; !equalDurations(got, want) {
		t.Fatalf("sleep durations = %v, want %v", got, want)
	}
	if got, want := logger.Events(), []string{"mine_failed"}; !equalStrings(got, want) {
		t.Fatalf("log events = %v, want %v", got, want)
	}
}

func TestSupervisorMinesRemainingWorkOnlyAfterCooldown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	enteredCooldown := make(chan time.Duration, 2)
	releaseCooldown := make(chan struct{})
	started := make(chan struct{}, 2)
	sleeper := &blockingSleeper{entered: enteredCooldown, release: releaseCooldown}
	miner := &fakeMiner{start: started}
	source := &scriptedStatusSource{results: []statusResult{{status: matchingStatus(2)}, {status: matchingStatus(1)}}}
	supervisor := newTestSupervisor(supervisorConfig(), source, miner, sleeper, nil)

	done := make(chan error, 1)
	go func() { done <- supervisor.Run(ctx) }()
	waitForSignal(t, started, "first MineOne call")
	if got := waitForDuration(t, enteredCooldown, "first cooldown"); got != 30*time.Second {
		t.Fatalf("first cooldown = %v, want 30s", got)
	}
	if got := miner.Calls(); got != 1 {
		t.Fatalf("MineOne calls before cooldown release = %d, want 1", got)
	}
	close(releaseCooldown)
	waitForSignal(t, started, "second MineOne call")
	cancel()
	err := waitForRun(t, done, "supervisor shutdown")

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if got := miner.Calls(); got != 2 {
		t.Fatalf("MineOne calls = %d, want 2", got)
	}
}

func TestSupervisorCancellationDuringSleepDoesNotPollAgain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sleeper := &recordingSleeper{after: func(int) { cancel() }}
	source := &scriptedStatusSource{results: []statusResult{{status: matchingStatus(0)}}}
	supervisor := newTestSupervisor(supervisorConfig(), source, &fakeMiner{}, sleeper, nil)

	err := supervisor.Run(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if got := source.Calls(); got != 1 {
		t.Fatalf("status calls = %d, want 1", got)
	}
}

func TestSupervisorCancellationAfterStatusDoesNotStartMiner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	miner := &fakeMiner{}
	source := &scriptedStatusSource{
		results: []statusResult{{status: matchingStatus(1)}},
		callHook: cancel,
	}
	supervisor := newTestSupervisor(supervisorConfig(), source, miner, &recordingSleeper{}, nil)

	err := supervisor.Run(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if got := miner.Calls(); got != 0 {
		t.Fatalf("MineOne calls = %d, want 0", got)
	}
}

func TestSupervisorRunsOnlyOneMineCallAtATime(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	miner := &fakeMiner{start: started, block: release}
	sleeper := &recordingSleeper{after: func(call int) {
		if call == 2 {
			cancel()
		}
	}}
	source := &scriptedStatusSource{results: []statusResult{{status: matchingStatus(1)}, {status: matchingStatus(1)}}}
	supervisor := newTestSupervisor(supervisorConfig(), source, miner, sleeper, nil)

	done := make(chan error, 1)
	go func() { done <- supervisor.Run(ctx) }()
	waitForSignal(t, started, "first MineOne call")
	if got := miner.MaxActive(); got != 1 {
		t.Fatalf("concurrent MineOne calls = %d, want 1", got)
	}
	if got := source.Calls(); got != 1 {
		t.Fatalf("status calls before first MineOne finishes = %d, want 1", got)
	}
	close(release)

	if err := waitForRun(t, done, "supervisor shutdown"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if got := miner.MaxActive(); got != 1 {
		t.Fatalf("maximum concurrent MineOne calls = %d, want 1", got)
	}
}

func waitForSignal(t *testing.T, signal <-chan struct{}, operation string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", operation)
	}
}

func waitForDuration(t *testing.T, signal <-chan time.Duration, operation string) time.Duration {
	t.Helper()
	select {
	case duration := <-signal:
		return duration
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", operation)
		return 0
	}
}

func waitForRun(t *testing.T, done <-chan error, operation string) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", operation)
		return nil
	}
}

func equalDurations(got, want []time.Duration) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
