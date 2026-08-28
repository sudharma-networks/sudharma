package demandminer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNativeRunnerUsesSafeUniqueCommandAndCleansOnlyRunDirectory(t *testing.T) {
	dataRoot := t.TempDir()
	sentinel := filepath.Join(dataRoot, "keep")
	if err := os.WriteFile(sentinel, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := nativeRunnerConfig(dataRoot)
	runner := NewNativeRunner(cfg)

	var nodeIDs []string
	var runDirs []string
	runner.ExecCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name != cfg.MinerBinary {
			t.Fatalf("binary = %q, want %q", name, cfg.MinerBinary)
		}
		if len(args) != 12 {
			t.Fatalf("arguments = %q, want 12 entries", args)
		}

		nodeID := args[1]
		runDir := args[7]
		nodeSuffix := strings.TrimPrefix(nodeID, "demand-miner-")
		runSuffix := strings.TrimPrefix(filepath.Base(runDir), "run-")
		if nodeSuffix == nodeID || nodeSuffix == "" || nodeSuffix != runSuffix {
			t.Fatalf("node ID %q and run directory %q do not share a unique suffix", nodeID, runDir)
		}
		want := []string{
			"-nodeid", "demand-miner-" + nodeSuffix,
			"-listen", "127.0.0.1:0",
			"-peer", "3.7.253.229:28444",
			"-datadir", filepath.Join(dataRoot, "run-"+nodeSuffix),
			"-mineblocks", "1",
			"-testmineraddress", "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
		}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("arguments = %q, want %q", args, want)
		}
		if info, err := os.Stat(runDir); err != nil || !info.IsDir() {
			t.Fatalf("ephemeral run directory was not created: info=%v err=%v", info, err)
		}
		if err := os.WriteFile(filepath.Join(runDir, "ephemeral"), []byte("remove"), 0600); err != nil {
			t.Fatal(err)
		}

		nodeIDs = append(nodeIDs, nodeID)
		runDirs = append(runDirs, runDir)
		return nativeRunnerHelperCommand(ctx, "success")
	}

	for i := 0; i < 2; i++ {
		if err := runner.MineOne(context.Background()); err != nil {
			t.Fatalf("MineOne call %d: %v", i+1, err)
		}
		if _, err := os.Stat(runDirs[i]); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("run directory %q still exists after MineOne: %v", runDirs[i], err)
		}
	}
	if nodeIDs[0] == nodeIDs[1] || runDirs[0] == runDirs[1] {
		t.Fatalf("successive runs reused identity or state: node IDs=%q run directories=%q", nodeIDs, runDirs)
	}
	if got, err := os.ReadFile(sentinel); err != nil || string(got) != "keep" {
		t.Fatalf("configured data-root content changed: contents=%q err=%v", got, err)
	}
	entries, err := os.ReadDir(dataRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "keep" {
		t.Fatalf("data root entries = %v, want only the pre-existing sentinel", entryNames(entries))
	}
}

func TestNativeRunnerRequiresPositivePendingAndIncludedTransactionEvidence(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want string
	}{
		{name: "missing pending count", mode: "missing-pending", want: "pending transaction"},
		{name: "zero pending count", mode: "zero-pending", want: "pending transaction"},
		{name: "missing included count", mode: "missing-included", want: "included transaction"},
		{name: "zero included count", mode: "zero-included", want: "included transaction"},
		{name: "unrelated transaction count", mode: "unrelated-included", want: "included transaction"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := NewNativeRunner(nativeRunnerConfig(t.TempDir()))
			runner.ExecCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				return nativeRunnerHelperCommand(ctx, test.mode)
			}

			err := runner.MineOne(context.Background())

			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("MineOne error = %v, want %q evidence rejection", err, test.want)
			}
		})
	}
}

func TestNativeRunnerRetainsAtMost64KiBOfCombinedOutput(t *testing.T) {
	runner := NewNativeRunner(nativeRunnerConfig(t.TempDir()))
	runner.ExecCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return nativeRunnerHelperCommand(ctx, "large-failure")
	}

	err := runner.MineOne(context.Background())

	if err == nil {
		t.Fatal("MineOne error = nil, want child-process failure")
	}
	if got := strings.Count(err.Error(), "~") + strings.Count(err.Error(), "^"); got == 0 || got > 64*1024 {
		t.Fatalf("retained output bytes = %d, want 1..65536", got)
	}
}

func TestNativeRunnerTerminatesOnChildTimeout(t *testing.T) {
	cfg := nativeRunnerConfig(t.TempDir())
	cfg.ChildTimeout = "1s"
	runner := NewNativeRunner(cfg)
	runner.ExecCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return nativeRunnerHelperCommand(ctx, "block")
	}

	started := time.Now()
	err := runner.MineOne(context.Background())

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("MineOne error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("timed-out child returned after %v, want at most 3s", elapsed)
	}
}

func TestNativeRunnerTerminatesOnCancellation(t *testing.T) {
	cfg := nativeRunnerConfig(t.TempDir())
	cfg.ChildTimeout = "10s"
	runner := NewNativeRunner(cfg)
	runner.ExecCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return nativeRunnerHelperCommand(ctx, "block")
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)

	started := time.Now()
	err := runner.MineOne(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("MineOne error = %v, want context canceled", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("canceled child returned after %v, want at most 2s", elapsed)
	}
}

func TestNativeRunnerRejectsInvalidConfigBeforeCreatingStateOrCommand(t *testing.T) {
	cfg := nativeRunnerConfig(t.TempDir())
	cfg.DataDirectory = "relative-data"
	runner := NewNativeRunner(cfg)
	called := false
	runner.ExecCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		called = true
		return nativeRunnerHelperCommand(ctx, "success")
	}

	err := runner.MineOne(context.Background())

	if err == nil || !strings.Contains(err.Error(), "invalid demand miner config") {
		t.Fatalf("MineOne error = %v, want invalid config rejection", err)
	}
	if called {
		t.Fatal("child command was created for invalid config")
	}
}

func TestNativeRunnerHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_NATIVE_RUNNER_HELPER") != "1" {
		return
	}
	switch os.Getenv("NATIVE_RUNNER_HELPER_MODE") {
	case "success":
		fmt.Fprintln(os.Stdout, "Pending Transactions: 2")
		fmt.Fprintln(os.Stderr, "Block #8 found | Hash: abc | Transactions: 2 | Reward: 1.00000000 SUDH | Work: 9")
	case "missing-pending":
		fmt.Fprintln(os.Stdout, "Block #8 found | Hash: abc | Transactions: 1 | Reward: 1.00000000 SUDH | Work: 9")
	case "zero-pending":
		fmt.Fprintln(os.Stdout, "Pending Transactions: 0")
		fmt.Fprintln(os.Stdout, "Block #8 found | Hash: abc | Transactions: 1 | Reward: 1.00000000 SUDH | Work: 9")
	case "missing-included":
		fmt.Fprintln(os.Stdout, "Pending Transactions: 1")
	case "zero-included":
		fmt.Fprintln(os.Stdout, "Pending Transactions: 1")
		fmt.Fprintln(os.Stdout, "Block #8 found | Hash: abc | Transactions: 0 | Reward: 1.00000000 SUDH | Work: 9")
	case "unrelated-included":
		fmt.Fprintln(os.Stdout, "Pending Transactions: 1")
		fmt.Fprintln(os.Stdout, "Transactions: 1")
	case "large-failure":
		fmt.Fprint(os.Stdout, strings.Repeat("~", 48*1024))
		fmt.Fprint(os.Stderr, strings.Repeat("^", 80*1024))
		os.Exit(9)
	case "block":
		time.Sleep(10 * time.Second)
	default:
		fmt.Fprintln(os.Stderr, "unknown helper mode")
		os.Exit(2)
	}
	os.Exit(0)
}

func nativeRunnerConfig(dataRoot string) Config {
	cfg := validConfig()
	cfg.SeedAddress = "3.7.253.229:28444"
	cfg.DataDirectory = dataRoot
	cfg.ChildTimeout = "5s"
	return cfg
}

func nativeRunnerHelperCommand(ctx context.Context, mode string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestNativeRunnerHelperProcess", "--")
	cmd.Env = append(os.Environ(),
		"GO_WANT_NATIVE_RUNNER_HELPER=1",
		"NATIVE_RUNNER_HELPER_MODE="+mode,
	)
	return cmd
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}
