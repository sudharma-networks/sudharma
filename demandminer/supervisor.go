package demandminer

import (
	"context"
	"fmt"
	"time"
)

// Status is the minimal public-testnet status information used for mining
// decisions. The supervisor deliberately does not receive transactions.
type Status struct {
	Network      string
	Coin         string
	Symbol       string
	Height       uint64
	IssuedSupply uint64
	Mempool      int
}

// StatusSource retrieves the current public-testnet status.
type StatusSource interface {
	Status(context.Context) (Status, error)
}

// BlockMiner creates and broadcasts one bounded block through the native miner.
type BlockMiner interface {
	MineOne(context.Context) error
}

// Sleeper waits between supervisor iterations while remaining cancellable.
type Sleeper interface {
	Sleep(context.Context, time.Duration) error
}

// Logger records recoverable operational failures without exposing supervisor
// configuration or transaction contents.
type Logger interface {
	Error(event string, fields map[string]any)
}

const (
	publicTestnetNetwork = "sudharma"
	publicTestnetCoin    = "Sudharma"
	publicTestnetSymbol  = "SUDH"
)

// Supervisor makes deterministic mining decisions from aggregate status only.
// It performs each MineOne call synchronously, so no more than one call can be
// active at a time.
type Supervisor struct {
	// Config is retained for operator inspection. Run uses the private,
	// validated config snapshot so later changes cannot affect mining decisions.
	Config Config

	config    Config
	configErr error

	Source  StatusSource
	Miner   BlockMiner
	Sleeper Sleeper
	Logger  Logger
}

// NewSupervisor constructs a supervisor from the ports needed by its loop. It
// snapshots and validates config so the public Config field cannot redirect a
// running supervisor to a different network identity.
func NewSupervisor(config Config, source StatusSource, miner BlockMiner, sleeper Sleeper, logger Logger) *Supervisor {
	return &Supervisor{
		Config:    config,
		config:    config,
		configErr: config.Validate(),
		Source:    source,
		Miner:     miner,
		Sleeper:   sleeper,
		Logger:    logger,
	}
}

// Run polls aggregate status and starts one bounded miner only when the
// validated public-testnet mempool is non-empty.
func (s *Supervisor) Run(ctx context.Context) error {
	if s.configErr != nil {
		return fmt.Errorf("invalid demand miner config: %w", s.configErr)
	}
	if err := s.validateDependencies(); err != nil {
		return err
	}
	pollEvery, err := s.config.PollDuration()
	if err != nil {
		return err
	}
	cooldown, err := s.config.CooldownDuration()
	if err != nil {
		return err
	}
	failureBackoff, err := s.config.FailureBackoffDuration()
	if err != nil {
		return err
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		status, err := s.Source.Status(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			s.logFailure("status_failed", err)
			if err := s.sleep(ctx, failureBackoff); err != nil {
				return err
			}
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.validateStatus(status); err != nil {
			return err
		}

		if status.Mempool == 0 {
			if err := s.sleep(ctx, pollEvery); err != nil {
				return err
			}
			continue
		}
		if status.Mempool < 0 {
			err := fmt.Errorf("invalid negative mempool count: %d", status.Mempool)
			s.logFailure("status_invalid", err)
			if err := s.sleep(ctx, failureBackoff); err != nil {
				return err
			}
			continue
		}

		if err := s.Miner.MineOne(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			s.logFailure("mine_failed", err)
			if err := s.sleep(ctx, failureBackoff); err != nil {
				return err
			}
			continue
		}
		if err := s.sleep(ctx, cooldown); err != nil {
			return err
		}
	}
}

func (s *Supervisor) validateDependencies() error {
	if s.Source == nil {
		return fmt.Errorf("status source is required")
	}
	if s.Miner == nil {
		return fmt.Errorf("block miner is required")
	}
	if s.Sleeper == nil {
		return fmt.Errorf("sleeper is required")
	}
	return nil
}

func (s *Supervisor) validateStatus(status Status) error {
	if status.Network != publicTestnetNetwork || status.Coin != publicTestnetCoin || status.Symbol != publicTestnetSymbol {
		return fmt.Errorf(
			"status identity mismatch: network=%q coin=%q symbol=%q",
			status.Network,
			status.Coin,
			status.Symbol,
		)
	}
	return nil
}

func (s *Supervisor) sleep(ctx context.Context, duration time.Duration) error {
	err := s.Sleeper.Sleep(ctx, duration)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func (s *Supervisor) logFailure(event string, err error) {
	if s.Logger != nil {
		s.Logger.Error(event, map[string]any{"error": err.Error()})
	}
}
