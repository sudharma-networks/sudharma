package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/demandminer"
	"github.com/sudharma-networks/sudharma/rpc"
)

func TestRunRequiresConfigFlag(t *testing.T) {
	err := runWithContext(context.Background(), nil, &bytes.Buffer{}, commandDependencies{})
	if err == nil || !strings.Contains(err.Error(), "-config is required") {
		t.Fatalf("run error = %v, want missing -config rejection", err)
	}
}

func TestRunRejectsInvalidConfigBeforeStartup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.json")
	cfg := commandConfig(t)
	cfg.Environment = "mainnet"
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	started := false
	deps := commandDependencies{
		newRPCClient: func(string) (rpcStatusClient, error) {
			started = true
			return nil, errors.New("RPC startup must not run")
		},
		newNativeRunner: func(demandminer.Config) demandminer.BlockMiner {
			started = true
			return nil
		},
	}
	err = runWithContext(context.Background(), []string{"-config", path}, &bytes.Buffer{}, deps)
	if err == nil || !strings.Contains(err.Error(), "environment must be public-testnet") {
		t.Fatalf("run error = %v, want invalid config rejection", err)
	}
	if started {
		t.Fatal("RPC client or miner started for invalid config")
	}
}

func TestRPCStatusSourceMapsAllMiningFields(t *testing.T) {
	source := rpcStatusSource{client: fakeRPCStatusClient{status: &rpc.NodeStatus{
		Network: "sudharma", Coin: "Sudharma", Symbol: "SUDH",
		Height: 12, Mempool: 3, IssuedSupply: 700,
	}}}

	got, err := source.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := demandminer.Status{
		Network: "sudharma", Coin: "Sudharma", Symbol: "SUDH",
		Height: 12, Mempool: 3, IssuedSupply: 700,
	}
	if got != want {
		t.Fatalf("mapped status = %+v, want %+v", got, want)
	}
}

func TestAcquireLockRejectsSecondProcess(t *testing.T) {
	first, err := acquireLock(filepath.Join(t.TempDir(), "miner.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if _, err := acquireLock(first.Name()); err == nil {
		t.Fatal("expected lock contention")
	}
}

func TestRunPassesCancellationToSupervisor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	supervisor := &waitingSupervisor{started: make(chan struct{})}
	deps := commandDependencies{
		loadConfig:      func(string) (demandminer.Config, error) { return commandConfig(t), nil },
		newRPCClient:    func(string) (rpcStatusClient, error) { return fakeRPCStatusClient{}, nil },
		newNativeRunner: func(demandminer.Config) demandminer.BlockMiner { return fakeBlockMiner{} },
		newSupervisor: func(demandminer.Config, demandminer.StatusSource, demandminer.BlockMiner, demandminer.Sleeper, demandminer.Logger) supervisorRunner {
			return supervisor
		},
	}

	errs := make(chan error, 1)
	go func() {
		errs <- runWithContext(ctx, []string{"-config", filepath.Join(t.TempDir(), "config.json")}, &bytes.Buffer{}, deps)
	}()
	select {
	case <-supervisor.started:
	case <-time.After(time.Second):
		t.Fatal("supervisor did not start")
	}
	cancel()
	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("run error = %v, want clean cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not reach supervisor")
	}
}

func TestRunEmitsSafeJSONLogs(t *testing.T) {
	var logs bytes.Buffer
	cfg := commandConfig(t)
	cfg.SeedAddress = "do-not-log-seed"
	deps := commandDependencies{
		loadConfig:      func(string) (demandminer.Config, error) { return cfg, nil },
		newRPCClient:    func(string) (rpcStatusClient, error) { return fakeRPCStatusClient{}, nil },
		newNativeRunner: func(demandminer.Config) demandminer.BlockMiner { return fakeBlockMiner{} },
		newSupervisor: func(demandminer.Config, demandminer.StatusSource, demandminer.BlockMiner, demandminer.Sleeper, demandminer.Logger) supervisorRunner {
			return completedSupervisor{err: context.Canceled}
		},
	}

	if err := runWithContext(context.Background(), []string{"-config", filepath.Join(t.TempDir(), "config.json")}, &logs, deps); err != nil {
		t.Fatalf("run error = %v", err)
	}
	if strings.Contains(logs.String(), "seed_address") || strings.Contains(logs.String(), "do-not-log-seed") {
		t.Fatalf("logs leaked configuration: %s", logs.String())
	}
	lines := strings.FieldsFunc(strings.TrimSpace(logs.String()), func(r rune) bool { return r == '\n' })
	if len(lines) < 2 {
		t.Fatalf("JSON log lines = %d, want at least 2: %s", len(lines), logs.String())
	}
	for _, line := range lines {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("invalid JSON log %q: %v", line, err)
		}
		for _, key := range []string{"time", "level", "event"} {
			if _, ok := record[key]; !ok {
				t.Fatalf("log record missing %q: %#v", key, record)
			}
		}
	}
}

type fakeRPCStatusClient struct {
	status *rpc.NodeStatus
	err    error
}

func (f fakeRPCStatusClient) Status(context.Context) (*rpc.NodeStatus, error) { return f.status, f.err }

type fakeBlockMiner struct{}

func (fakeBlockMiner) MineOne(context.Context) error { return nil }

type waitingSupervisor struct{ started chan struct{} }

func (s *waitingSupervisor) Run(ctx context.Context) error {
	close(s.started)
	<-ctx.Done()
	return ctx.Err()
}

type completedSupervisor struct{ err error }

func (s completedSupervisor) Run(context.Context) error { return s.err }

func commandConfig(t *testing.T) demandminer.Config {
	t.Helper()
	dir := t.TempDir()
	return demandminer.Config{
		Environment: "public-testnet", StatusURL: "http://127.0.0.1:28545",
		ExpectedNetwork: "sudharma", ExpectedCoin: "Sudharma", ExpectedSymbol: "SUDH",
		SeedAddress: "127.0.0.1:28444", RewardAddress: "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
		MinerBinary: "/bin/true", DataDirectory: dir, LockFile: filepath.Join(dir, "miner.lock"),
		PollEvery: "10s", Cooldown: "30s", FailureBackoff: "30s", ChildTimeout: "5s",
	}
}
