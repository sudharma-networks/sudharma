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
	config    Config
	configErr error
	status    StatusSource
	balances  RewardBalanceSource
	miner     BlockMiner
	sleeper   Sleeper
	logger    Logger
	now       func() time.Time
}

func NewSupervisor(config Config, status StatusSource, miner BlockMiner, sleeper Sleeper, logger Logger, balances RewardBalanceSource) *Supervisor {
	return &Supervisor{
		config:    config,
		configErr: config.Validate(),
		status:    status,
		balances:  balances,
		miner:     miner,
		sleeper:   sleeper,
		logger:    logger,
		now:       time.Now,
	}
}

func (s *Supervisor) Run(ctx context.Context) error {
	if s.configErr != nil {
		return fmt.Errorf("invalid demand miner config: %w", s.configErr)
	}
	lastSweep := s.now()
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

		if status.Mempool < 0 {
			err := fmt.Errorf("invalid negative mempool count: %d", status.Mempool)
			s.logError("status_invalid", err)
			if err := s.sleeper.Sleep(ctx, s.config.FailureBackoffDuration()); err != nil {
				return err
			}
			continue
		}

		if status.Mempool > 0 {
			if err := s.clearPendingMempool(ctx); err != nil {
				s.logError("sweep_error", err)
				if err := s.sleeper.Sleep(ctx, s.config.FailureBackoffDuration()); err != nil {
					return err
				}
				continue
			}
			lastSweep = s.now()
			if err := s.sleeper.Sleep(ctx, s.config.CooldownDuration()); err != nil {
				return err
			}
			continue
		}

		needsFunding, err := s.needsFaucetFunding(ctx)
		if err != nil {
			s.logError("faucet_funding_status_error", err)
			if err := s.sleeper.Sleep(ctx, s.config.FailureBackoffDuration()); err != nil {
				return err
			}
			continue
		}
		if needsFunding {
			if err := s.mineFaucetFundingBlocks(ctx); err != nil {
				s.logError("faucet_funding_error", err)
				if err := s.sleeper.Sleep(ctx, s.config.FailureBackoffDuration()); err != nil {
					return err
				}
				continue
			}
			lastSweep = s.now()
			if err := s.sleeper.Sleep(ctx, s.config.CooldownDuration()); err != nil {
				return err
			}
			continue
		}

		scheduledDue := !s.now().Before(lastSweep.Add(s.config.ScheduledSweepDuration()))
		if scheduledDue {
			if err := s.miner.MineOne(ctx); err != nil {
				s.logError("scheduled_reward_error", err)
				if err := s.sleeper.Sleep(ctx, s.config.FailureBackoffDuration()); err != nil {
					return err
				}
				continue
			}
			lastSweep = s.now()
			if err := s.sleeper.Sleep(ctx, s.config.CooldownDuration()); err != nil {
				return err
			}
			continue
		}

		if err := s.sleeper.Sleep(ctx, s.config.PollDuration()); err != nil {
			return err
		}
	}
}

func (s *Supervisor) clearPendingMempool(ctx context.Context) error {
	limit := s.config.BlocksPerSweepLimit()
	for mined := 0; mined < limit; mined++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		status, err := s.status.Status(ctx)
		if err != nil {
			return err
		}
		if err := s.validateStatusIdentity(status); err != nil {
			return err
		}
		if status.Mempool <= 0 {
			return nil
		}

		if err := s.miner.MineOne(ctx); err != nil {
			return err
		}
	}

	status, err := s.status.Status(ctx)
	if err != nil {
		return err
	}
	if status.Mempool > 0 {
		return fmt.Errorf("mempool still has %d transactions after %d blocks", status.Mempool, limit)
	}
	return nil
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
