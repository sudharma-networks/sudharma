// sudharma-demand-miner runs the bounded native miner only when the public
// testnet RPC reports pending transactions.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/sudharma-networks/sudharma/demandminer"
	"github.com/sudharma-networks/sudharma/operations"
	"github.com/sudharma-networks/sudharma/rpc"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "sudharma-demand-miner: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runWithContext(ctx, args, output, commandDependencies{})
}

type rpcStatusClient interface {
	Status(context.Context) (*rpc.NodeStatus, error)
}

type supervisorRunner interface {
	Run(context.Context) error
}

type commandDependencies struct {
	loadConfig      func(string) (demandminer.Config, error)
	newRPCClient    func(string) (rpcStatusClient, error)
	newNativeRunner func(demandminer.Config) demandminer.BlockMiner
	newSupervisor   func(demandminer.Config, demandminer.StatusSource, demandminer.BlockMiner, demandminer.Sleeper, demandminer.Logger) supervisorRunner
	acquireLock     func(string) (*os.File, error)
}

func (d commandDependencies) withDefaults() commandDependencies {
	if d.loadConfig == nil {
		d.loadConfig = demandminer.LoadConfig
	}
	if d.newRPCClient == nil {
		d.newRPCClient = func(url string) (rpcStatusClient, error) { return rpc.NewClient(url) }
	}
	if d.newNativeRunner == nil {
		d.newNativeRunner = func(cfg demandminer.Config) demandminer.BlockMiner { return demandminer.NewNativeRunner(cfg) }
	}
	if d.newSupervisor == nil {
		d.newSupervisor = func(cfg demandminer.Config, source demandminer.StatusSource, miner demandminer.BlockMiner, sleeper demandminer.Sleeper, logger demandminer.Logger) supervisorRunner {
			return demandminer.NewSupervisor(cfg, source, miner, sleeper, logger)
		}
	}
	if d.acquireLock == nil {
		d.acquireLock = acquireLock
	}
	return d
}

func runWithContext(ctx context.Context, args []string, output io.Writer, dependencies commandDependencies) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	if output == nil {
		output = io.Discard
	}
	flags := flag.NewFlagSet("sudharma-demand-miner", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "absolute demand miner JSON config path")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *configPath == "" {
		return fmt.Errorf("-config is required")
	}
	if !filepath.IsAbs(*configPath) {
		return fmt.Errorf("-config must be an absolute path")
	}

	deps := dependencies.withDefaults()
	cfg, err := deps.loadConfig(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid demand miner config: %w", err)
	}

	log := operations.NewLogger(output, true)
	lock, err := deps.acquireLock(cfg.LockFile)
	if err != nil {
		log.Error("demand_miner_lock_failed", nil)
		return fmt.Errorf("acquire demand miner lock: %w", err)
	}
	defer lock.Close()

	client, err := deps.newRPCClient(cfg.StatusURL)
	if err != nil {
		log.Error("demand_miner_rpc_client_failed", nil)
		return fmt.Errorf("create RPC client: %w", err)
	}
	if client == nil {
		log.Error("demand_miner_rpc_client_failed", nil)
		return fmt.Errorf("create RPC client: client is required")
	}
	miner := deps.newNativeRunner(cfg)
	if miner == nil {
		log.Error("demand_miner_runner_failed", nil)
		return fmt.Errorf("create native runner: runner is required")
	}
	supervisor := deps.newSupervisor(cfg, rpcStatusSource{client: client}, miner, contextSleeper{}, eventLogger{logger: log})
	if supervisor == nil {
		log.Error("demand_miner_supervisor_failed", nil)
		return fmt.Errorf("create supervisor: supervisor is required")
	}

	log.Info("demand_miner_started", nil)
	err = supervisor.Run(ctx)
	if errors.Is(err, context.Canceled) {
		log.Info("demand_miner_stopped", nil)
		return nil
	}
	if err != nil {
		log.Error("demand_miner_stopped", nil)
		return fmt.Errorf("run supervisor: %w", err)
	}
	log.Info("demand_miner_stopped", nil)
	return nil
}

type rpcStatusSource struct{ client rpcStatusClient }

func (s rpcStatusSource) Status(ctx context.Context) (demandminer.Status, error) {
	if s.client == nil {
		return demandminer.Status{}, fmt.Errorf("RPC status client is required")
	}
	status, err := s.client.Status(ctx)
	if err != nil {
		return demandminer.Status{}, err
	}
	if status == nil {
		return demandminer.Status{}, fmt.Errorf("RPC status response is required")
	}
	return demandminer.Status{
		Network:      status.Network,
		Coin:         status.Coin,
		Symbol:       status.Symbol,
		Height:       status.Height,
		Mempool:      status.Mempool,
		IssuedSupply: status.IssuedSupply,
	}, nil
}

type contextSleeper struct{}

func (contextSleeper) Sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type eventLogger struct{ logger *operations.Logger }

func (l eventLogger) Error(event string, _ map[string]any) {
	if l.logger != nil {
		l.logger.Error(event, nil)
	}
}

func acquireLock(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lock file: %w", err)
	}
	return file, nil
}
