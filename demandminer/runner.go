package demandminer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const maxNativeMinerOutput = 64 * 1024

// ExecCommandContextFunc constructs a child process bound to a context.
type ExecCommandContextFunc func(context.Context, string, ...string) *exec.Cmd

// NativeRunner invokes the existing native miner for one transaction-bearing
// block. It snapshots validated configuration when constructed so later
// caller mutation cannot redirect its binary, peer, reward, or state path.
type NativeRunner struct {
	config             Config
	configErr          error
	ExecCommandContext ExecCommandContextFunc
}

var _ BlockMiner = (*NativeRunner)(nil)

// NewNativeRunner constructs a bounded native miner process runner.
func NewNativeRunner(config Config) *NativeRunner {
	return &NativeRunner{
		config:             config,
		configErr:          config.Validate(),
		ExecCommandContext: exec.CommandContext,
	}
}

// MineOne starts one uniquely identified native miner, requires evidence that
// pending transactions were included, and removes only that run's state.
func (r *NativeRunner) MineOne(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("native runner is required")
	}
	if r.configErr != nil {
		return fmt.Errorf("invalid demand miner config: %w", r.configErr)
	}
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.ExecCommandContext == nil {
		return fmt.Errorf("exec command context is required")
	}

	timeout, err := r.config.ChildTimeoutDuration()
	if err != nil {
		return err
	}
	runDir, suffix, err := createVerifiedRunDirectory(r.config.DataDirectory)
	if err != nil {
		return err
	}
	defer os.RemoveAll(runDir)

	childCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := []string{
		"-nodeid", "demand-miner-" + suffix,
		"-listen", "127.0.0.1:0",
		"-peer", r.config.SeedAddress,
		"-datadir", runDir,
		"-mineblocks", "1",
		"-testmineraddress", r.config.RewardAddress,
	}
	cmd := r.ExecCommandContext(childCtx, r.config.MinerBinary, args...)
	if cmd == nil {
		return fmt.Errorf("exec command context returned a nil command")
	}

	output := &boundedOutput{limit: maxNativeMinerOutput}
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		if childErr := childCtx.Err(); childErr != nil {
			return childErr
		}
		return fmt.Errorf("native miner failed: %w; output: %s", err, output.String())
	}
	if childErr := childCtx.Err(); childErr != nil {
		return childErr
	}

	pending, included := nativeMinerEvidence(output.String())
	if !pending {
		return fmt.Errorf("native miner output lacks positive pending transaction evidence; output: %s", output.String())
	}
	if !included {
		return fmt.Errorf("native miner output lacks positive included transaction evidence; output: %s", output.String())
	}
	return nil
}

func createVerifiedRunDirectory(dataRoot string) (string, string, error) {
	runDir, err := os.MkdirTemp(dataRoot, "run-")
	if err != nil {
		return "", "", fmt.Errorf("create native miner run directory: %w", err)
	}
	cleanRoot := filepath.Clean(dataRoot)
	cleanRunDir := filepath.Clean(runDir)
	base := filepath.Base(cleanRunDir)
	suffix := strings.TrimPrefix(base, "run-")
	if filepath.Dir(cleanRunDir) != cleanRoot || suffix == base || suffix == "" {
		return "", "", fmt.Errorf("refusing unsafe native miner run directory %q", runDir)
	}
	return cleanRunDir, suffix, nil
}

func nativeMinerEvidence(output string) (pending bool, included bool) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if positiveCount(line, "Pending Transactions:") {
			pending = true
		}
		if !strings.HasPrefix(line, "Block #") {
			continue
		}
		for _, field := range strings.Split(line, "|") {
			if positiveCount(strings.TrimSpace(field), "Transactions:") {
				included = true
			}
		}
	}
	return pending, included
}

func positiveCount(value, prefix string) bool {
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	count, err := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(value, prefix)), 10, 64)
	return err == nil && count > 0
}

type boundedOutput struct {
	mu    sync.Mutex
	data  []byte
	limit int
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - len(b.data)
	if remaining > 0 {
		if remaining > len(p) {
			remaining = len(p)
		}
		b.data = append(b.data, p[:remaining]...)
	}
	return len(p), nil
}

func (b *boundedOutput) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.data)
}
