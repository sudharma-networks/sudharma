package blockchain

import (
	"math"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestMaximumSupplyEnforced(t *testing.T) {
	state := NewState()

	// Mint exactly the maximum Sudharma Network supply.
	err := state.MintSupply(params.MaxSupply)

	if err != nil {
		t.Fatalf(
			"minting max supply should succeed: %v",
			err,
		)
	}

	if state.IssuedSupply() != params.MaxSupply {
		t.Fatalf(
			"expected issued supply %d, got %d",
			params.MaxSupply,
			state.IssuedSupply(),
		)
	}

	// Even one additional base unit must fail.
	err = state.MintSupply(1)

	if err == nil {
		t.Fatal(
			"minting above maximum supply was accepted",
		)
	}

	// Supply must remain unchanged after failed mint.
	if state.IssuedSupply() != params.MaxSupply {
		t.Fatalf(
			"issued supply changed after rejected mint: %d",
			state.IssuedSupply(),
		)
	}
}

func TestMintCannotExceedRemainingSupply(t *testing.T) {
	state := NewState()

	almostMaximum := params.MaxSupply - 10

	if err := state.MintSupply(almostMaximum); err != nil {
		t.Fatal(err)
	}

	// Only 10 base units remain.
	if err := state.MintSupply(11); err == nil {
		t.Fatal(
			"mint larger than remaining supply was accepted",
		)
	}

	if state.IssuedSupply() != almostMaximum {
		t.Fatal(
			"issued supply changed after rejected mint",
		)
	}

	// Exactly the remaining 10 units should succeed.
	if err := state.MintSupply(10); err != nil {
		t.Fatal(err)
	}

	if state.IssuedSupply() != params.MaxSupply {
		t.Fatalf(
			"expected maximum supply %d, got %d",
			params.MaxSupply,
			state.IssuedSupply(),
		)
	}
}

func TestBalanceOverflowRejected(t *testing.T) {
	state := NewState()

	address := "overflow-test-address"

	if err := state.Credit(
		address,
		math.MaxUint64,
	); err != nil {
		t.Fatal(err)
	}

	// Adding even one more unit would overflow uint64.
	err := state.Credit(address, 1)

	if err == nil {
		t.Fatal(
			"balance overflow was accepted",
		)
	}

	// Balance must remain unchanged.
	if state.Balance(address) != math.MaxUint64 {
		t.Fatalf(
			"balance changed after rejected overflow: %d",
			state.Balance(address),
		)
	}
}

func TestMintSupplyForMainnetEnforcesMainnetCap(t *testing.T) {
	state := NewState()

	if err := state.MintSupplyFor(params.MonetaryPolicyMainnet, params.MainnetMaxSupply); err != nil {
		t.Fatal(err)
	}
	if state.IssuedSupply() != params.MainnetMaxSupply {
		t.Fatalf("expected issued supply %d, got %d", params.MainnetMaxSupply, state.IssuedSupply())
	}

	if err := state.MintSupplyFor(params.MonetaryPolicyMainnet, 1); err == nil {
		t.Fatal("mainnet mint above 51M was accepted")
	}
	if state.IssuedSupply() != params.MainnetMaxSupply {
		t.Fatalf("issued supply changed after rejected mainnet mint: %d", state.IssuedSupply())
	}

	// Public-testnet MintSupply still uses the 51B ceiling.
	if err := state.MintSupply(1); err != nil {
		t.Fatalf("testnet mint under 51B should succeed: %v", err)
	}
	if state.IssuedSupply() != params.MainnetMaxSupply+1 {
		t.Fatalf("expected issued supply %d, got %d", params.MainnetMaxSupply+1, state.IssuedSupply())
	}
}

func TestMintSupplyCompatibilityWrapperKeepsTestnetCap(t *testing.T) {
	state := NewState()
	if err := state.MintSupply(params.MaxSupply); err != nil {
		t.Fatal(err)
	}
	if err := state.MintSupply(1); err == nil {
		t.Fatal("testnet mint above 51B was accepted")
	}
}

func TestMintSupplyForRejectsUnknownPolicy(t *testing.T) {
	state := NewState()
	if err := state.MintSupplyFor(params.MonetaryPolicy(255), 1); err == nil {
		t.Fatal("expected unknown monetary policy error")
	}
	if state.IssuedSupply() != 0 {
		t.Fatal("unknown policy mutated issued supply")
	}
}

func TestCreditMinerRewardForStopsAtMainnetCap(t *testing.T) {
	state := NewState()
	remaining := uint64(1)
	alreadyIssued := params.MainnetMaxSupply - remaining
	if err := state.MintSupplyFor(params.MonetaryPolicyMainnet, alreadyIssued); err != nil {
		t.Fatal(err)
	}

	got, err := CreditMinerRewardFor(state, params.MonetaryPolicyMainnet, 1, "miner", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != remaining {
		t.Fatalf("expected clipped mainnet subsidy %d, got %d", remaining, got)
	}
	if state.IssuedSupply() != params.MainnetMaxSupply {
		t.Fatalf("expected issued supply %d, got %d", params.MainnetMaxSupply, state.IssuedSupply())
	}

	second, err := CreditMinerRewardFor(state, params.MonetaryPolicyMainnet, 2, "miner", 9_000)
	if err != nil {
		t.Fatal(err)
	}
	if second != 9_000 {
		t.Fatalf("expected fee-only reward 9000, got %d", second)
	}
	if state.IssuedSupply() != params.MainnetMaxSupply {
		t.Fatal("fee-only mainnet block minted new supply")
	}
}

func TestMinerSubsidyStopsAtMaxSupply(t *testing.T) {
	state := NewState()

	minerAddress := "supply-cap-miner"

	// Leave only 5 SUDH available for future issuance.
	remaining := uint64(5 * params.CoinDecimals)

	alreadyIssued :=
		params.MaxSupply - remaining

	if err := state.MintSupply(alreadyIssued); err != nil {
		t.Fatal(err)
	}

	// Normal subsidy would be 50 SUDH,
	// but only 5 SUDH remain under the cap.
	reward, err := CreditMinerReward(
		state,
		1,
		minerAddress,
		0,
	)

	if err != nil {
		t.Fatal(err)
	}

	if reward != remaining {
		t.Fatalf(
			"expected final subsidy %d, got %d",
			remaining,
			reward,
		)
	}

	if state.IssuedSupply() != params.MaxSupply {
		t.Fatalf(
			"expected issued supply %d, got %d",
			params.MaxSupply,
			state.IssuedSupply(),
		)
	}

	if state.Balance(minerAddress) != remaining {
		t.Fatalf(
			"expected miner balance %d, got %d",
			remaining,
			state.Balance(minerAddress),
		)
	}

	// Once max supply has been reached,
	// another block should create no new subsidy.
	secondReward, err := CreditMinerReward(
		state,
		2,
		minerAddress,
		0,
	)

	if err != nil {
		t.Fatal(err)
	}

	if secondReward != 0 {
		t.Fatalf(
			"expected zero subsidy after max supply, got %d",
			secondReward,
		)
	}

	if state.IssuedSupply() != params.MaxSupply {
		t.Fatal(
			"issued supply exceeded Sudharma Network maximum",
		)
	}
}
