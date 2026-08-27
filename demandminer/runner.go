package demandminer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const maxMinerOutputBytes = 64 << 10

var (
	pendingTransactionsPattern = regexp.MustCompile(`(?m)^Pending Transactions:\s*([0-9]+)\s*$`)
	includedTransactionsPattern = regexp.MustCompile(`(?m)^Transactions:\s*([0-9]+)\s*$`)
)

type ExecCommandContext func(context.Context, string, ...string) *exec.Cmd

type NativeRunner struct {
	config  Config
	command ExecCommandContext
}

func NewNativeRunner(config Config, command ExecCommandContext) *NativeRunner {
	if command == nil {
		command = exec.CommandContext
	}
	return &NativeRunner{config: config, command: command}
}

func (r *NativeRunner) MineOne(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(r.config.DataDirectory, 0o700); err != nil {
		return fmt.Errorf("create miner data directory: %w", err)
	}

	runDir, err := os.MkdirTemp(r.config.DataDirectory, "run-")
	if err != nil {
		return fmt.Errorf("create miner run directory: %w", err)
	}
	if !isImmediateChild(r.config.DataDirectory, runDir) {
		return errors.New("refusing unsafe miner run directory")
	}
	defer os.RemoveAll(runDir)

	runName := filepath.Base(runDir)
	nodeID := "demand-miner-" + strings.TrimPrefix(runName, "run-")
	args := []string{
		"-nodeid", nodeID,
		"-listen", "127.0.0.1:0",
		"-peer", r.config.SeedAddress,
		"-datadir", runDir,
		"-mineblocks", "1",
		"-testmineraddress", r.config.RewardAddress,
	}

	runCtx, cancel := context.WithTimeout(ctx, r.config.ChildTimeoutDuration())
	defer cancel()
	cmd := r.command(runCtx, r.config.MinerBinary, args...)
	var output boundedBuffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	runErr := cmd.Run()
	if err := runCtx.Err(); err != nil {
		return err
	}
	if runErr != nil {
		return fmt.Errorf("miner child failed: %w; output: %s", runErr, output.String())
	}
	if err := validateMiningOutput(output.String()); err != nil {
		return err
	}
	return nil
}

func validateMiningOutput(output string) error {
	pending, ok := parseCount(pendingTransactionsPattern, output)
	if !ok {
		return errors.New("miner output missing Pending Transactions evidence")
	}
	if pending <= 0 {
		return errors.New("miner reported no pending transactions")
	}
	included, ok := parseCount(includedTransactionsPattern, output)
	if !ok {
		return errors.New("miner output missing Transactions evidence")
	}
	if included <= 0 {
		return errors.New("miner reported no included transactions")
	}
	return nil
}

func parseCount(pattern *regexp.Regexp, output string) (int64, bool) {
	match := pattern.FindStringSubmatch(output)
	if len(match) != 2 {
		return 0, false
	}
	value, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func isImmediateChild(parent, child string) bool {
	parentAbs, err := filepath.Abs(parent)
	if err != nil {
		return false
	}
	childAbs, err := filepath.Abs(child)
	if err != nil {
		return false
	}
	return filepath.Dir(childAbs) == filepath.Clean(parentAbs) && childAbs != parentAbs
}

type boundedBuffer struct {
	buf bytes.Buffer
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remaining := maxMinerOutputBytes - b.buf.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.buf.Write(p)
	}
	return original, nil
}

func (b *boundedBuffer) String() string { return b.buf.String() }

var _ io.Writer = (*boundedBuffer)(nil)
var _ BlockMiner = (*NativeRunner)(nil)
