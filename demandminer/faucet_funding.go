package demandminer

import (
	"context"
	"errors"
)

var ErrFaucetFundingIncomplete = errors.New("faucet funding incomplete after configured block limit")

type RewardBalanceSource interface {
	RewardBalance(context.Context) (uint64, error)
}

func (s *Supervisor) rewardBalance(ctx context.Context) (uint64, error) {
	if s.balances == nil {
		return 0, nil
	}
	return s.balances.RewardBalance(ctx)
}

func (s *Supervisor) needsFaucetFunding(ctx context.Context) (bool, error) {
	if s.balances == nil || s.config.FaucetMinBalance == 0 {
		return false, nil
	}
	balance, err := s.rewardBalance(ctx)
	if err != nil {
		return false, err
	}
	return balance < s.config.FaucetMinBalance, nil
}

func (s *Supervisor) mineFaucetFundingBlocks(ctx context.Context) error {
	if s.balances == nil || s.config.FaucetMinBalance == 0 {
		return nil
	}
	blocks := s.config.FaucetFundingBlocksLimit()
	for mined := 0; mined < blocks; mined++ {
		balance, err := s.rewardBalance(ctx)
		if err != nil {
			return err
		}
		if balance >= s.config.FaucetMinBalance {
			return nil
		}
		if err := s.miner.MineOne(ctx); err != nil {
			return err
		}
	}
	balance, err := s.rewardBalance(ctx)
	if err != nil {
		return err
	}
	if balance < s.config.FaucetMinBalance {
		return ErrFaucetFundingIncomplete
	}
	return nil
}
