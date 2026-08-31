package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/params"
)

func TestCreditMinerReward(t *testing.T) {
	state := NewState()
	minerAddress := "sudharma-test-miner"
	minerFees := uint64(9_000_000)

	totalReward, err := CreditMinerReward(state, 1, minerAddress, minerFees)
	if err != nil {
		t.Fatal(err)
	}

	expected := consensus.BlockSubsidy(1) + minerFees
	if totalReward != expected {
		t.Fatalf("expected reward %d, got %d", expected, totalReward)
	}
	if state.Balance(minerAddress) != expected {
		t.Fatalf("expected miner balance %d, got %d", expected, state.Balance(minerAddress))
	}
}

func TestMinerRewardAfterFirstHalving(t *testing.T) {
	state := NewState()
	minerAddress := "halving-test-miner"

	totalReward, err := CreditMinerReward(state, params.HalvingInterval, minerAddress, 0)
	if err != nil {
		t.Fatal(err)
	}

	expected := uint64(25 * params.CoinDecimals)
	if totalReward != expected {
		t.Fatalf("expected 25 SUDH reward, got %d base units", totalReward)
	}
}

func TestMinerRewardRequiresAddress(t *testing.T) {
	state := NewState()
	_, err := CreditMinerReward(state, 1, "", 0)
	if err == nil {
		t.Fatal("expected empty miner address error")
	}
}

func TestCreditMinerRewardForMainnetUsesMainnetSubsidyAndMinerFees(t *testing.T) {
	state := NewState()
	fees := uint64(12_345)
	wantSubsidy, err := consensus.BlockSubsidyFor(params.MonetaryPolicyMainnet, 1)
	if err != nil {
		t.Fatal(err)
	}

	got, err := CreditMinerRewardFor(state, params.MonetaryPolicyMainnet, 1, "miner", fees)
	if err != nil {
		t.Fatal(err)
	}
	if got != wantSubsidy+fees {
		t.Fatalf("expected %d got %d", wantSubsidy+fees, got)
	}
	if state.Balance("miner") != wantSubsidy+fees {
		t.Fatalf("expected miner balance %d got %d", wantSubsidy+fees, state.Balance("miner"))
	}
	if state.IssuedSupply() != wantSubsidy {
		t.Fatalf("fees must not mint supply: issued=%d subsidy=%d", state.IssuedSupply(), wantSubsidy)
	}
}

func TestCreditMinerRewardCompatibilityWrapperKeepsTestnetReward(t *testing.T) {
	state := NewState()
	got, err := CreditMinerReward(state, 0, "miner", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != 50*params.CoinDecimals {
		t.Fatalf("testnet reward changed: got %d", got)
	}
}
