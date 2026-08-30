package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/sudharma-networks/sudharma/demandminer"
	"github.com/sudharma-networks/sudharma/rpc"
)

type statusClient interface {
	Status(context.Context) (*rpc.NodeStatus, error)
}

type rpcStatusSource struct {
	client statusClient
}

type rpcRewardBalanceSource struct {
	client  *rpc.Client
	address string
}

func (s rpcRewardBalanceSource) RewardBalance(ctx context.Context) (uint64, error) {
	if s.client == nil {
		return 0, errors.New("RPC client is unavailable")
	}
	account, err := s.client.Account(ctx, s.address)
	if err != nil {
		return 0, err
	}
	if account == nil {
		return 0, errors.New("RPC returned an empty account")
	}
	return account.Balance, nil
}

func (s rpcStatusSource) Status(ctx context.Context) (demandminer.Status, error) {
	if s.client == nil {
		return demandminer.Status{}, errors.New("RPC status client is unavailable")
	}
	status, err := s.client.Status(ctx)
	if err != nil {
		return demandminer.Status{}, err
	}
	if status == nil {
		return demandminer.Status{}, errors.New("RPC returned an empty status")
	}
	return demandminer.Status{
		Network:      status.Network,
		Coin:         status.Coin,
		Symbol:       status.Symbol,
		Height:       status.Height,
		IssuedSupply: status.IssuedSupply,
		Mempool:      status.Mempool,
	}, nil
}

type processLock struct {
	file *os.File
}

func acquireLock(path string) (*processLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("acquire lock: %w", err)
	}
	return &processLock{file: file}, nil
}

func (l *processLock) Name() string {
	if l == nil || l.file == nil {
		return ""
	}
	return l.file.Name()
}

func (l *processLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	unlockErr := unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	closeErr := l.file.Close()
	l.file = nil
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

type jsonLogger struct {
	encoder *json.Encoder
}

func newJSONLogger(w io.Writer) *jsonLogger {
	if w == nil {
		w = io.Discard
	}
	return &jsonLogger{encoder: json.NewEncoder(w)}
}

func (l *jsonLogger) Error(event string, err error) {
	if l == nil || l.encoder == nil {
		return
	}
	message := ""
	if err != nil {
		message = err.Error()
	}
	_ = l.encoder.Encode(struct {
		Event string `json:"event"`
		Error string `json:"error"`
	}{Event: event, Error: message})
}

func run(ctx context.Context, args []string, logWriter io.Writer) error {
	flags := flag.NewFlagSet("sudharma-demand-miner", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "absolute path to demand-miner JSON configuration")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *configPath == "" {
		return errors.New("-config is required")
	}
	if !filepath.IsAbs(*configPath) {
		return errors.New("-config must be an absolute path")
	}

	cfg, err := demandminer.LoadConfig(*configPath)
	if err != nil {
		return fmt.Errorf("load demand miner config: %w", err)
	}

	lock, err := acquireLock(cfg.LockFile)
	if err != nil {
		return err
	}
	defer lock.Close()

	client, err := rpc.NewClient(cfg.StatusURL)
	if err != nil {
		return fmt.Errorf("create RPC client: %w", err)
	}

	logger := newJSONLogger(logWriter)
	runner := demandminer.NewNativeRunner(cfg, nil)
	supervisor := demandminer.NewSupervisor(
		cfg,
		rpcStatusSource{client: client},
		runner,
		demandminer.TimerSleeper{},
		logger,
		rpcRewardBalanceSource{client: client, address: cfg.RewardAddress},
	)
	return supervisor.Run(ctx)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
