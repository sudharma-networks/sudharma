package demandminer

import (
	"context"
	"fmt"
	"time"
)

type Status struct {
	Network      string
	Coin         string
	Symbol       string
	Height       uint64
	IssuedSupply uint64
	Mempool      int
}

type StatusSource interface {
	Status(context.Context) (Status, error)
}

type BlockMiner interface {
	MineOne(context.Context) error
}

type Sleeper interface {
	Sleep(context.Context, time.Duration) error
}

type Logger interface {
	Error(event string, err error)
}

type TimerSleeper struct{}

func (TimerSleeper) Sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

type Supervisor struct {
	config  Config
	status  StatusSource
	miner   BlockMiner
	sleeper Sleeper
	logger  Logger
}

func NewSupervisor(config Config, status StatusSource, miner BlockMiner, sleeper Sleeper, logger Logger) *Supervisor {
	return &Supervisor{config: config, status: status, miner: miner, sleeper: sleeper, logger: logger}
}

func (s *Supervisor) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		status, err := s.status.Status(ctx)
		if err != nil {
			s.logError("status_error", err)
			if err := s.sleeper.Sleep(ctx, s.config.FailureBackoffDuration()); err != nil {
				return err
			}
			continue
		}
		if err := s.validateStatusIdentity(status); err != nil {
			return err
		}

		if status.Mempool <= 0 {
			if err := s.sleeper.Sleep(ctx, s.config.PollDuration()); err != nil {
				return err
			}
			continue
		}

		if err := s.miner.MineOne(ctx); err != nil {
			s.logError("mine_error", err)
			if err := s.sleeper.Sleep(ctx, s.config.FailureBackoffDuration()); err != nil {
				return err
			}
			continue
		}
		if err := s.sleeper.Sleep(ctx, s.config.CooldownDuration()); err != nil {
			return err
		}
	}
}

func (s *Supervisor) validateStatusIdentity(status Status) error {
	if status.Network != s.config.ExpectedNetwork || status.Coin != s.config.ExpectedCoin || status.Symbol != s.config.ExpectedSymbol {
		return fmt.Errorf("status identity mismatch: network=%q coin=%q symbol=%q", status.Network, status.Coin, status.Symbol)
	}
	return nil
}

func (s *Supervisor) logError(event string, err error) {
	if s.logger != nil {
		s.logger.Error(event, err)
	}
}
