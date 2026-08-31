package demandminer

import (
	"context"
	"errors"
	"testing"
)

type fakeRewardBalanceSource struct {
	balances []uint64
	calls    int
	err      error
}

func (f *fakeRewardBalanceSource) RewardBalance(context.Context) (uint64, error) {
	f.calls++
	if f.err != nil {
		return 0, f.err
	}
	if len(f.balances) == 0 {
		return 0, nil
	}
	balance := f.balances[0]
	if len(f.balances) > 1 {
		f.balances = f.balances[1:]
	}
	return balance, nil
}

func TestNeedsFaucetFundingDisabledWithoutBalanceSource(t *testing.T) {
	s := NewSupervisor(validConfig(), &fakeStatusSource{}, &fakeMiner{}, &stopSleeper{}, &fakeLogger{}, nil)
	needs, err := s.needsFaucetFunding(context.Background())
	if err != nil {
		t.Fatalf("needsFaucetFunding: %v", err)
	}
	if needs {
		t.Fatal("expected faucet funding to be disabled without balance source")
	}
}

func TestMineFaucetFundingBlocksMinesUntilFunded(t *testing.T) {
	cfg := validConfig()
	cfg.FaucetMinBalance = 100
	cfg.FaucetFundingBlocks = 2
	balances := &fakeRewardBalanceSource{balances: []uint64{10, 10, 120}}
	miner := &fakeMiner{}
	s := NewSupervisor(cfg, &fakeStatusSource{}, miner, &stopSleeper{}, &fakeLogger{}, balances)

	if err := s.mineFaucetFundingBlocks(context.Background()); err != nil {
		t.Fatalf("mineFaucetFundingBlocks: %v", err)
	}
	if miner.calls != 2 {
		t.Fatalf("MineOne calls = %d", miner.calls)
	}
}

func TestMineFaucetFundingBlocksReturnsIncompleteError(t *testing.T) {
	cfg := validConfig()
	cfg.FaucetMinBalance = 100
	cfg.FaucetFundingBlocks = 2
	balances := &fakeRewardBalanceSource{balances: []uint64{10, 10, 10, 10}}
	miner := &fakeMiner{}
	s := NewSupervisor(cfg, &fakeStatusSource{}, miner, &stopSleeper{}, &fakeLogger{}, balances)

	err := s.mineFaucetFundingBlocks(context.Background())
	if !errors.Is(err, ErrFaucetFundingIncomplete) {
		t.Fatalf("expected ErrFaucetFundingIncomplete, got %v", err)
	}
	if miner.calls != 2 {
		t.Fatalf("MineOne calls = %d", miner.calls)
	}
}

func TestSupervisorPrioritizesFaucetFundingOverScheduledSweep(t *testing.T) {
	cfg := validConfig()
	cfg.FaucetMinBalance = 100
	cfg.FaucetFundingBlocks = 2
	cfg.ScheduledSweepEvery = "30m"
	balances := &fakeRewardBalanceSource{balances: []uint64{10, 10, 10, 120, 120}}
	source := &fakeStatusSource{results: []statusResult{
		{status: validStatus(0)},
		{status: validStatus(0)},
	}}
	miner := &fakeMiner{}
	sleeper := &stopSleeper{stopAfter: 1}
	s := newTestSupervisor(cfg, source, miner, sleeper, &fakeLogger{}, nil, balances)

	if err := s.Run(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run: %v", err)
	}
	if miner.calls != 2 {
		t.Fatalf("MineOne calls = %d", miner.calls)
	}
}

func TestConfigFaucetFundingBlocksLimitDefaultsToTwo(t *testing.T) {
	cfg := validConfig()
	if got := cfg.FaucetFundingBlocksLimit(); got != 2 {
		t.Fatalf("FaucetFundingBlocksLimit() = %d", got)
	}
	cfg.FaucetFundingBlocks = 5
	if got := cfg.FaucetFundingBlocksLimit(); got != 5 {
		t.Fatalf("FaucetFundingBlocksLimit() = %d", got)
	}
}
