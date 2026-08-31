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
	"sync"
)

const (
	maxMinerOutputBytes  = 64 << 10
	miningEvidenceWindow = 8 << 10
)

var (
	pendingTransactionsPattern  = regexp.MustCompile(`(?m)^Pending Transactions:\s*([0-9]+)\s*$`)
	includedTransactionsPattern = regexp.MustCompile(`(?m)^Block #[0-9]+ found \| Hash: [^|]+ \| Transactions:\s*([0-9]+)\s*\|`)
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

	runCtx, cancelRun := context.WithTimeout(ctx, r.config.ChildTimeoutDuration())
	defer cancelRun()
	childCtx, stopChild := context.WithCancel(runCtx)
	defer stopChild()

	evidence := make(chan struct{})
	var evidenceOnce sync.Once
	output := miningOutput{onEvidence: func() {
		evidenceOnce.Do(func() { close(evidence) })
	}}

	cmd := r.command(childCtx, r.config.MinerBinary, args...)
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start miner child: %w", err)
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	select {
	case runErr := <-waitCh:
		if err := runCtx.Err(); err != nil {
			return err
		}
		if runErr != nil {
			return fmt.Errorf("miner child failed: %w; output: %s", runErr, output.String())
		}
		if err := output.validationError(); err != nil {
			return err
		}
		return nil

	case <-evidence:
		if err := runCtx.Err(); err != nil {
			stopChild()
			<-waitCh
			return err
		}

		// sudharmad enters its normal node loop after -mineblocks completes.
		// The included-transaction evidence is emitted only after the mined
		// block has been successfully broadcast. At that point this runner
		// stops only the unique ephemeral child it created, then reaps it.
		stopChild()
		<-waitCh
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := output.validationError(); err != nil {
			return err
		}
		return nil

	case <-runCtx.Done():
		stopChild()
		<-waitCh
		return runCtx.Err()
	}
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

type miningOutput struct {
	mu sync.Mutex

	retained bytes.Buffer
	tail     []byte

	pendingSeen      bool
	pendingPositive  bool
	includedSeen     bool
	includedPositive bool

	onEvidence func()
}

func (o *miningOutput) Write(p []byte) (int, error) {
	original := len(p)
	o.mu.Lock()

	remaining := maxMinerOutputBytes - o.retained.Len()
	if remaining > 0 {
		chunk := p
		if len(chunk) > remaining {
			chunk = chunk[:remaining]
		}
		_, _ = o.retained.Write(chunk)
	}

	o.appendEvidenceTail(p)
	o.observeEvidence(string(o.tail))
	complete := o.pendingPositive && o.includedPositive
	onEvidence := o.onEvidence
	o.mu.Unlock()

	if complete && onEvidence != nil {
		onEvidence()
	}
	return original, nil
}

func (o *miningOutput) appendEvidenceTail(p []byte) {
	if len(p) >= miningEvidenceWindow {
		o.tail = append(o.tail[:0], p[len(p)-miningEvidenceWindow:]...)
		return
	}
	overflow := len(o.tail) + len(p) - miningEvidenceWindow
	if overflow > 0 {
		copy(o.tail, o.tail[overflow:])
		o.tail = o.tail[:len(o.tail)-overflow]
	}
	o.tail = append(o.tail, p...)
}

func (o *miningOutput) observeEvidence(text string) {
	for _, match := range pendingTransactionsPattern.FindAllStringSubmatch(text, -1) {
		if len(match) != 2 {
			continue
		}
		value, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			continue
		}
		o.pendingSeen = true
		if value > 0 {
			o.pendingPositive = true
		}
	}
	for _, match := range includedTransactionsPattern.FindAllStringSubmatch(text, -1) {
		if len(match) != 2 {
			continue
		}
		value, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			continue
		}
		o.includedSeen = true
		if value > 0 {
			o.includedPositive = true
		}
	}
}

func (o *miningOutput) validationError() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.pendingSeen {
		return errors.New("miner output missing Pending Transactions evidence")
	}
	if !o.pendingPositive {
		return errors.New("miner reported no pending transactions")
	}
	if !o.includedSeen {
		return errors.New("miner output missing Transactions evidence")
	}
	if !o.includedPositive {
		return errors.New("miner reported no included transactions")
	}
	return nil
}

func (o *miningOutput) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.retained.String()
}

var _ io.Writer = (*miningOutput)(nil)
var _ BlockMiner = (*NativeRunner)(nil)
