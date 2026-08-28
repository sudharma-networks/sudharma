package demandminer

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNativeRunnerUsesExactBoundedMiningArguments(t *testing.T) {
	cfg := validConfig()
	cfg.SeedAddress = "3.7.253.229:28444"
	cfg.DataDirectory = t.TempDir()
	cfg.ChildTimeout = "5s"

	var gotName string
	var gotArgs []string
	runner := NewNativeRunner(cfg, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return helperCommand(ctx, "success")
	})

	if err := runner.MineOne(context.Background()); err != nil {
		t.Fatalf("MineOne: %v", err)
	}
	if gotName != cfg.MinerBinary {
		t.Fatalf("binary = %q, want %q", gotName, cfg.MinerBinary)
	}
	if len(gotArgs) != 12 {
		t.Fatalf("args = %v", gotArgs)
	}
	if gotArgs[0] != "-nodeid" || !strings.HasPrefix(gotArgs[1], "demand-miner-") {
		t.Fatalf("node args = %v", gotArgs[:2])
	}
	if gotArgs[2] != "-listen" || gotArgs[3] != "127.0.0.1:0" || gotArgs[4] != "-peer" || gotArgs[5] != cfg.SeedAddress {
		t.Fatalf("network args = %v", gotArgs[2:6])
	}
	if gotArgs[6] != "-datadir" || filepath.Dir(gotArgs[7]) != cfg.DataDirectory || !strings.HasPrefix(filepath.Base(gotArgs[7]), "run-") {
		t.Fatalf("datadir args = %v", gotArgs[6:8])
	}
	wantTail := []string{"-mineblocks", "1", "-testmineraddress", cfg.RewardAddress}
	if !reflect.DeepEqual(gotArgs[8:], wantTail) {
		t.Fatalf("tail args = %v, want %v", gotArgs[8:], wantTail)
	}
	if _, err := os.Stat(gotArgs[7]); !os.IsNotExist(err) {
		t.Fatalf("run directory must be removed, stat err = %v", err)
	}
}

func TestNativeRunnerStopsOwnChildAfterBroadcastEvidence(t *testing.T) {
	runner := testRunner(t, "long-running-success", "30s")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := runner.MineOne(ctx); err != nil {
		t.Fatalf("MineOne must stop its own long-running child after mining evidence, before parent deadline: %v", err)
	}
}

func TestNativeRunnerDetectsEvidenceAfterRetainedOutputCap(t *testing.T) {
	runner := testRunner(t, "late-evidence", "5s")
	if err := runner.MineOne(context.Background()); err != nil {
		t.Fatalf("MineOne must detect mining evidence even after retained diagnostic output is full: %v", err)
	}
}

func TestNativeRunnerRejectsOutputWithoutPendingEvidence(t *testing.T) {
	runner := testRunner(t, "missing-pending", "5s")
	if err := runner.MineOne(context.Background()); err == nil || !strings.Contains(err.Error(), "Pending Transactions") {
		t.Fatalf("expected pending evidence error, got %v", err)
	}
}

func TestNativeRunnerRejectsOutputWithoutIncludedTransactionEvidence(t *testing.T) {
	runner := testRunner(t, "missing-included", "5s")
	if err := runner.MineOne(context.Background()); err == nil || !strings.Contains(err.Error(), "Transactions") {
		t.Fatalf("expected transaction evidence error, got %v", err)
	}
}

func TestNativeRunnerRejectsZeroPendingTransactions(t *testing.T) {
	runner := testRunner(t, "zero-pending", "5s")
	if err := runner.MineOne(context.Background()); err == nil || !strings.Contains(err.Error(), "no pending") {
		t.Fatalf("expected no-pending error, got %v", err)
	}
}

func TestNativeRunnerBoundsRetainedOutput(t *testing.T) {
	runner := testRunner(t, "huge-failure", "5s")
	err := runner.MineOne(context.Background())
	if err == nil {
		t.Fatal("expected child failure")
	}
	if len(err.Error()) > maxMinerOutputBytes+1024 {
		t.Fatalf("error retained too much output: %d bytes", len(err.Error()))
	}
}

func TestNativeRunnerTerminatesOnTimeout(t *testing.T) {
	runner := testRunner(t, "sleep", "100ms")
	start := time.Now()
	err := runner.MineOne(context.Background())
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline error, got %v", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("timeout was not bounded")
	}
}

func TestNativeRunnerHonorsParentCancellation(t *testing.T) {
	runner := testRunner(t, "sleep", "5s")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runner.MineOne(ctx); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func testRunner(t *testing.T, mode, timeout string) *NativeRunner {
	t.Helper()
	cfg := validConfig()
	cfg.DataDirectory = t.TempDir()
	cfg.ChildTimeout = timeout
	return NewNativeRunner(cfg, func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return helperCommand(ctx, mode)
	})
}

func helperCommand(ctx context.Context, mode string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestNativeRunnerHelperProcess", "--", mode)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

func TestNativeRunnerHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	mode := args[len(args)-1]
	switch mode {
	case "success":
		_, _ = os.Stdout.WriteString("Pending Transactions: 2\nBlock #1 found | Hash: abc | Transactions: 2 | Reward: 50.00000000 SUDH | Work: 2\n")
	case "long-running-success":
		_, _ = os.Stdout.WriteString("Pending Transactions: 1\nBlock #1 found | Hash: abc | Transactions: 1 | Reward: 50.00000000 SUDH | Work: 2\n")
		time.Sleep(10 * time.Second)
	case "late-evidence":
		_, _ = os.Stdout.WriteString(strings.Repeat("x", maxMinerOutputBytes*2))
		_, _ = os.Stdout.WriteString("\nPending Transactions: 1\nBlock #1 found | Hash: abc | Transactions: 1 | Reward: 50.00000000 SUDH | Work: 2\n")
	case "missing-pending":
		_, _ = os.Stdout.WriteString("Block #1 found | Hash: abc | Transactions: 1 | Reward: 50.00000000 SUDH | Work: 2\n")
	case "missing-included":
		_, _ = os.Stdout.WriteString("Pending Transactions: 1\n")
	case "zero-pending":
		_, _ = os.Stdout.WriteString("Pending Transactions: 0\nBlock #1 found | Hash: abc | Transactions: 0 | Reward: 50.00000000 SUDH | Work: 2\n")
	case "huge-failure":
		_, _ = os.Stdout.WriteString(strings.Repeat("x", maxMinerOutputBytes*2))
		os.Exit(2)
	case "sleep":
		time.Sleep(10 * time.Second)
	}
	os.Exit(0)
}