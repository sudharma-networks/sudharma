# Sudharma Mainnet Tokenomics v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a mainnet-only 51,000,000 SUDH hard cap with the approved 40-quarter, 10-year declining subsidy schedule while preserving current public-testnet economics and fee behavior.

**Architecture:** Introduce explicit monetary-policy selection instead of allowing `consensus.BlockSubsidy` and `blockchain.CreditMinerReward` to read one global reward schedule. Keep the existing public-testnet policy unchanged, add a deterministic mainnet subsidy table in integer base units, and thread the selected policy through block reward application. Consensus tests prove every epoch boundary, exact annual totals, exact 51M cap, and zero subsidy after height 5,259,600.

**Tech Stack:** Go, existing `params`, `consensus`, and `blockchain` packages, standard `testing` package, GitHub Actions CI.

**Spec:** `docs/superpowers/specs/2026-08-31-mainnet-tokenomics-v1-design.md`

## Global Constraints

- Mainnet maximum monetary supply: exactly `51_000_000 * 100_000_000 = 5_100_000_000_000_000` base units.
- Premine: `0`.
- Target block time: `60` seconds.
- Mainnet subsidy-bearing heights: `1..5_259_600`; all later heights return zero subsidy.
- 40 subsidy epochs; every epoch is exactly `131_490` blocks.
- Transaction fee remains `0.10%`: `0.09%` miner + `0.01%` development treasury.
- Treasury remains a normal spendable wallet; ordinary transfers to/from treasury stay valid.
- Existing public testnet reward schedule remains unchanged while this work is implemented and tested.
- Reward selection is deterministic by block height and network monetary policy only; never by wall-clock time.
- Integer base-unit arithmetic only; no floating-point consensus arithmetic.
- No live testnet deployment, chain reset, faucet change, treasury movement, or mainnet activation in this plan.

---

## File Structure

- `params/monetary.go` — defines `MonetaryPolicy`, public-testnet monetary constants, and approved mainnet monetary constants.
- `params/mainnet_emission.go` — stores the 40 immutable mainnet epoch targets as integer base-unit values.
- `params/params.go` — retains network identity, fee constants, treasury address, decimals, and compatibility aliases needed by existing testnet code while migration is in progress.
- `consensus/rewards.go` — exposes policy-aware deterministic subsidy calculation.
- `consensus/rewards_test.go` — proves legacy testnet subsidy behavior plus all mainnet epoch boundaries and final cap properties.
- `blockchain/rewards.go` — credits subsidy using the explicitly selected monetary policy and enforces that policy's cap.
- `blockchain/rewards_test.go` — proves miner subsidy + fees, exact cap behavior, and mainnet/testnet isolation.
- `blockchain/block_processor.go` — accepts/uses the selected monetary policy when applying block rewards.
- `blockchain/block_processor_test.go` — proves end-to-end block accounting for both policies.

---

### Task 1: Introduce explicit monetary-policy types and constants

**Files:**
- Create: `params/monetary.go`
- Modify: `params/params.go`
- Test: `consensus/rewards_test.go`

**Interfaces:**
- Produces: `type MonetaryPolicy uint8`
- Produces: `const MonetaryPolicyPublicTestnet MonetaryPolicy = 1`
- Produces: `const MonetaryPolicyMainnet MonetaryPolicy = 2`
- Produces: `func MaxSupplyFor(policy MonetaryPolicy) uint64`
- Produces: `func ValidateMonetaryPolicy(policy MonetaryPolicy) error`

- [ ] **Step 1: Write failing policy-constant tests**

Add to `consensus/rewards_test.go`:

```go
func TestMonetaryPolicySupplyCaps(t *testing.T) {
	if got := params.MaxSupplyFor(params.MonetaryPolicyMainnet); got != 5_100_000_000_000_000 {
		t.Fatalf("mainnet max supply: expected %d, got %d", uint64(5_100_000_000_000_000), got)
	}
	if got := params.MaxSupplyFor(params.MonetaryPolicyPublicTestnet); got != params.MaxSupply {
		t.Fatalf("testnet max supply changed: expected %d, got %d", params.MaxSupply, got)
	}
}

func TestValidateMonetaryPolicyRejectsUnknown(t *testing.T) {
	if err := params.ValidateMonetaryPolicy(params.MonetaryPolicy(255)); err == nil {
		t.Fatal("expected unknown monetary policy to be rejected")
	}
}
```

- [ ] **Step 2: Run the focused test and confirm RED**

Run:

```bash
go test ./consensus -run 'TestMonetaryPolicySupplyCaps|TestValidateMonetaryPolicyRejectsUnknown' -count=1
```

Expected: compile failure because `MonetaryPolicy`, `MaxSupplyFor`, and `ValidateMonetaryPolicy` do not exist.

- [ ] **Step 3: Implement monetary-policy definitions**

Create `params/monetary.go`:

```go
package params

import "fmt"

type MonetaryPolicy uint8

const (
	MonetaryPolicyPublicTestnet MonetaryPolicy = 1
	MonetaryPolicyMainnet       MonetaryPolicy = 2

	MainnetMaxSupplySUDH uint64 = 51_000_000
	MainnetMaxSupply            = MainnetMaxSupplySUDH * CoinDecimals
	MainnetFinalSubsidyHeight   uint64 = 5_259_600
	MainnetEpochLength          uint64 = 131_490
	MainnetEpochCount           uint64 = 40
)

func ValidateMonetaryPolicy(policy MonetaryPolicy) error {
	switch policy {
	case MonetaryPolicyPublicTestnet, MonetaryPolicyMainnet:
		return nil
	default:
		return fmt.Errorf("unknown monetary policy %d", policy)
	}
}

func MaxSupplyFor(policy MonetaryPolicy) uint64 {
	switch policy {
	case MonetaryPolicyMainnet:
		return MainnetMaxSupply
	case MonetaryPolicyPublicTestnet:
		return MaxSupply
	default:
		return 0
	}
}
```

Keep current `MaxSupply`, `InitialBlockReward`, and `HalvingInterval` in `params/params.go` unchanged so existing public-testnet callers retain their behavior during migration.

- [ ] **Step 4: Run focused tests and full params/consensus tests**

```bash
go test ./params ./consensus -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add params/monetary.go params/params.go consensus/rewards_test.go
git commit -m "feat(consensus): define network monetary policies"
```

---

### Task 2: Encode the exact 40-epoch mainnet issuance table

**Files:**
- Create: `params/mainnet_emission.go`
- Test: `consensus/rewards_test.go`

**Interfaces:**
- Produces: `type EmissionEpoch struct { Issuance uint64 }`
- Produces: `var MainnetEmissionEpochs [40]EmissionEpoch`
- Produces immutable annual totals equivalent to the approved spec.

- [ ] **Step 1: Write failing emission-table invariant tests**

Add:

```go
func TestMainnetEmissionTableInvariants(t *testing.T) {
	if len(params.MainnetEmissionEpochs) != 40 {
		t.Fatalf("expected 40 epochs, got %d", len(params.MainnetEmissionEpochs))
	}

	var total uint64
	for _, epoch := range params.MainnetEmissionEpochs {
		total += epoch.Issuance
	}
	if total != params.MainnetMaxSupply {
		t.Fatalf("expected exact 51M issuance, got %d base units", total)
	}

	yearTargets := []uint64{
		8_160_000, 7_140_000, 6_630_000, 6_120_000, 5_610_000,
		5_100_000, 4_080_000, 3_570_000, 2_550_000, 2_040_000,
	}
	for year, wantSUDH := range yearTargets {
		var got uint64
		for q := 0; q < 4; q++ {
			got += params.MainnetEmissionEpochs[year*4+q].Issuance
		}
		want := wantSUDH * params.CoinDecimals
		if got != want {
			t.Fatalf("year %d: expected %d, got %d", year+1, want, got)
		}
	}
}
```

- [ ] **Step 2: Run and confirm RED**

```bash
go test ./consensus -run TestMainnetEmissionTableInvariants -count=1
```

Expected: compile failure because `MainnetEmissionEpochs` does not exist.

- [ ] **Step 3: Add exact integer epoch targets from the approved spec**

Create `params/mainnet_emission.go`:

```go
package params

type EmissionEpoch struct {
	Issuance uint64
}

var MainnetEmissionEpochs = [40]EmissionEpoch{
	{2_101_200 * CoinDecimals}, {2_060_400 * CoinDecimals}, {2_019_600 * CoinDecimals}, {1_978_800 * CoinDecimals},
	{1_838_550 * CoinDecimals}, {1_802_850 * CoinDecimals}, {1_767_150 * CoinDecimals}, {1_731_450 * CoinDecimals},
	{1_707_225 * CoinDecimals}, {1_674_075 * CoinDecimals}, {1_640_925 * CoinDecimals}, {1_607_775 * CoinDecimals},
	{1_575_900 * CoinDecimals}, {1_545_300 * CoinDecimals}, {1_514_700 * CoinDecimals}, {1_484_100 * CoinDecimals},
	{1_444_575 * CoinDecimals}, {1_416_525 * CoinDecimals}, {1_388_475 * CoinDecimals}, {1_360_425 * CoinDecimals},
	{1_313_250 * CoinDecimals}, {1_287_750 * CoinDecimals}, {1_262_250 * CoinDecimals}, {1_236_750 * CoinDecimals},
	{1_050_600 * CoinDecimals}, {1_030_200 * CoinDecimals}, {1_009_800 * CoinDecimals}, {989_400 * CoinDecimals},
	{919_275 * CoinDecimals}, {901_425 * CoinDecimals}, {883_575 * CoinDecimals}, {865_725 * CoinDecimals},
	{656_625 * CoinDecimals}, {643_875 * CoinDecimals}, {631_125 * CoinDecimals}, {618_375 * CoinDecimals},
	{525_300 * CoinDecimals}, {515_100 * CoinDecimals}, {504_900 * CoinDecimals}, {494_700 * CoinDecimals},
}
```

- [ ] **Step 4: Run the invariant tests**

```bash
go test ./consensus -run TestMainnetEmissionTableInvariants -count=1
```

Expected: PASS with exact annual and 51M totals.

- [ ] **Step 5: Commit**

```bash
git add params/mainnet_emission.go consensus/rewards_test.go
git commit -m "feat(consensus): encode mainnet emission epochs"
```

---

### Task 3: Implement deterministic policy-aware block subsidy

**Files:**
- Modify: `consensus/rewards.go`
- Modify: `consensus/rewards_test.go`

**Interfaces:**
- Produces: `func BlockSubsidyFor(policy params.MonetaryPolicy, height uint64) (uint64, error)`
- Preserves: `func BlockSubsidy(height uint64) uint64` as the public-testnet compatibility wrapper.

- [ ] **Step 1: Add failing exact-boundary tests**

Add tests that iterate all 40 epochs and verify first/last block reward plus the first block after final emission:

```go
func TestMainnetSubsidyEpochBoundaries(t *testing.T) {
	for epochIndex, epoch := range params.MainnetEmissionEpochs {
		start := uint64(epochIndex)*params.MainnetEpochLength + 1
		end := start + params.MainnetEpochLength - 1
		base := epoch.Issuance / params.MainnetEpochLength
		remainder := epoch.Issuance % params.MainnetEpochLength

		firstWant := base
		if remainder > 0 {
			firstWant++
		}
		first, err := BlockSubsidyFor(params.MonetaryPolicyMainnet, start)
		if err != nil || first != firstWant {
			t.Fatalf("epoch %d first block: want %d got %d err=%v", epochIndex+1, firstWant, first, err)
		}

		lastOffset := params.MainnetEpochLength - 1
		lastWant := base
		if lastOffset < remainder {
			lastWant++
		}
		last, err := BlockSubsidyFor(params.MonetaryPolicyMainnet, end)
		if err != nil || last != lastWant {
			t.Fatalf("epoch %d last block: want %d got %d err=%v", epochIndex+1, lastWant, last, err)
		}
	}

	got, err := BlockSubsidyFor(params.MonetaryPolicyMainnet, params.MainnetFinalSubsidyHeight+1)
	if err != nil || got != 0 {
		t.Fatalf("post-emission subsidy: want 0 got %d err=%v", got, err)
	}
}
```

Also add an exhaustive cumulative test over all subsidy-bearing heights:

```go
func TestMainnetCumulativeSubsidyIsExactHardCap(t *testing.T) {
	var total uint64
	for height := uint64(1); height <= params.MainnetFinalSubsidyHeight; height++ {
		reward, err := BlockSubsidyFor(params.MonetaryPolicyMainnet, height)
		if err != nil {
			t.Fatal(err)
		}
		if total > ^uint64(0)-reward {
			t.Fatal("subsidy total overflow")
		}
		total += reward
	}
	if total != params.MainnetMaxSupply {
		t.Fatalf("expected %d, got %d", params.MainnetMaxSupply, total)
	}
}
```

- [ ] **Step 2: Run and confirm RED**

```bash
go test ./consensus -run 'TestMainnetSubsidyEpochBoundaries|TestMainnetCumulativeSubsidyIsExactHardCap' -count=1
```

Expected: compile failure because `BlockSubsidyFor` does not exist.

- [ ] **Step 3: Implement policy-aware subsidy using integer division and remainder**

Replace `consensus/rewards.go` with logic equivalent to:

```go
package consensus

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/params"
)

func BlockSubsidy(height uint64) uint64 {
	reward, _ := BlockSubsidyFor(params.MonetaryPolicyPublicTestnet, height)
	return reward
}

func BlockSubsidyFor(policy params.MonetaryPolicy, height uint64) (uint64, error) {
	if err := params.ValidateMonetaryPolicy(policy); err != nil {
		return 0, err
	}
	if policy == params.MonetaryPolicyPublicTestnet {
		halvings := height / params.HalvingInterval
		if halvings >= 64 {
			return 0, nil
		}
		return params.InitialBlockReward >> halvings, nil
	}
	if height == 0 || height > params.MainnetFinalSubsidyHeight {
		return 0, nil
	}

	epochIndex := (height - 1) / params.MainnetEpochLength
	if epochIndex >= uint64(len(params.MainnetEmissionEpochs)) {
		return 0, fmt.Errorf("mainnet epoch index %d out of range", epochIndex)
	}
	epoch := params.MainnetEmissionEpochs[epochIndex]
	base := epoch.Issuance / params.MainnetEpochLength
	remainder := epoch.Issuance % params.MainnetEpochLength
	offset := (height - 1) % params.MainnetEpochLength
	if offset < remainder {
		return base + 1, nil
	}
	return base, nil
}
```

- [ ] **Step 4: Run consensus tests**

```bash
go test ./consensus -count=1
```

Expected: PASS, including unchanged legacy public-testnet halving tests.

- [ ] **Step 5: Commit**

```bash
git add consensus/rewards.go consensus/rewards_test.go
git commit -m "feat(consensus): add exact mainnet subsidy schedule"
```

---

### Task 4: Make miner reward accounting policy-aware and cap-safe

**Files:**
- Modify: `blockchain/rewards.go`
- Create or Modify: `blockchain/rewards_test.go`

**Interfaces:**
- Produces: `func CreditMinerRewardFor(state *State, policy params.MonetaryPolicy, blockHeight uint64, minerAddress string, minerFees uint64) (uint64, error)`
- Preserves: `CreditMinerReward(...)` as a public-testnet compatibility wrapper until all existing callers migrate.
- Produces: `func stateRemainingSupplyFor(state *State, policy params.MonetaryPolicy) uint64`

- [ ] **Step 1: Write failing accounting tests**

Add tests for mainnet and testnet isolation:

```go
func TestCreditMinerRewardForMainnetUsesMainnetSubsidyAndMinerFees(t *testing.T) {
	state := NewState()
	fees := uint64(12345)
	wantSubsidy, err := consensus.BlockSubsidyFor(params.MonetaryPolicyMainnet, 1)
	if err != nil { t.Fatal(err) }

	got, err := CreditMinerRewardFor(state, params.MonetaryPolicyMainnet, 1, "miner", fees)
	if err != nil { t.Fatal(err) }
	if got != wantSubsidy+fees {
		t.Fatalf("expected %d got %d", wantSubsidy+fees, got)
	}
	if state.IssuedSupply() != wantSubsidy {
		t.Fatalf("fees must not mint supply: issued=%d subsidy=%d", state.IssuedSupply(), wantSubsidy)
	}
}

func TestCreditMinerRewardCompatibilityWrapperKeepsTestnetReward(t *testing.T) {
	state := NewState()
	got, err := CreditMinerReward(state, 0, "miner", 0)
	if err != nil { t.Fatal(err) }
	if got != 50*params.CoinDecimals {
		t.Fatalf("testnet reward changed: got %d", got)
	}
}
```

- [ ] **Step 2: Run and confirm RED**

```bash
go test ./blockchain -run 'TestCreditMinerRewardForMainnet|TestCreditMinerRewardCompatibilityWrapper' -count=1
```

Expected: compile failure because `CreditMinerRewardFor` does not exist.

- [ ] **Step 3: Implement policy-aware reward credit**

Refactor `blockchain/rewards.go` so:

```go
func CreditMinerReward(
	state *State,
	blockHeight uint64,
	minerAddress string,
	minerFees uint64,
) (uint64, error) {
	return CreditMinerRewardFor(
		state,
		params.MonetaryPolicyPublicTestnet,
		blockHeight,
		minerAddress,
		minerFees,
	)
}
```

and `CreditMinerRewardFor` obtains subsidy from:

```go
subsidy, err := consensus.BlockSubsidyFor(policy, blockHeight)
if err != nil {
	return 0, err
}
```

Supply clipping must use:

```go
remaining := stateRemainingSupplyFor(state, policy)
```

with:

```go
func stateRemainingSupplyFor(state *State, policy params.MonetaryPolicy) uint64 {
	maxSupply := params.MaxSupplyFor(policy)
	issued := state.IssuedSupply()
	if issued >= maxSupply {
		return 0
	}
	return maxSupply - issued
}
```

Do not change fee accounting: only `subsidy` calls `MintSupply`; `minerFees` are credited but do not increase issued supply.

- [ ] **Step 4: Run blockchain reward tests**

```bash
go test ./blockchain -run 'Reward|Supply' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add blockchain/rewards.go blockchain/rewards_test.go
git commit -m "feat(blockchain): apply network-specific monetary caps"
```

---

### Task 5: Thread monetary policy through block processing without changing public testnet behavior

**Files:**
- Modify: `blockchain/block_processor.go`
- Modify: `blockchain/block_processor_test.go`

**Interfaces:**
- Produces: `func ProcessBlockFor(state *State, policy params.MonetaryPolicy, block *Block, minerAddress string) (uint64, error)`
- Preserves: `ProcessBlock(...)` as a public-testnet compatibility wrapper.

- [ ] **Step 1: Add failing end-to-end policy-isolation tests**

Add a mainnet block-processing test using an empty block at height 1:

```go
func TestProcessBlockForMainnetCreditsMainnetSubsidy(t *testing.T) {
	state := NewState()
	block := &Block{Height: 1}
	want, err := consensus.BlockSubsidyFor(params.MonetaryPolicyMainnet, 1)
	if err != nil { t.Fatal(err) }

	got, err := ProcessBlockFor(state, params.MonetaryPolicyMainnet, block, "miner")
	if err != nil { t.Fatal(err) }
	if got != want {
		t.Fatalf("expected %d got %d", want, got)
	}
	if state.IssuedSupply() != want {
		t.Fatalf("expected issued supply %d got %d", want, state.IssuedSupply())
	}
}

func TestProcessBlockCompatibilityWrapperStillUsesPublicTestnet(t *testing.T) {
	state := NewState()
	block := &Block{Height: 0}
	got, err := ProcessBlock(state, block, "miner")
	if err != nil { t.Fatal(err) }
	if got != 50*params.CoinDecimals {
		t.Fatalf("public testnet behavior changed: got %d", got)
	}
}
```

- [ ] **Step 2: Run and confirm RED**

```bash
go test ./blockchain -run 'TestProcessBlockForMainnet|TestProcessBlockCompatibilityWrapper' -count=1
```

Expected: compile failure because `ProcessBlockFor` does not exist.

- [ ] **Step 3: Implement policy-aware processing**

Make `ProcessBlock` delegate to public testnet:

```go
func ProcessBlock(state *State, block *Block, minerAddress string) (uint64, error) {
	return ProcessBlockFor(state, params.MonetaryPolicyPublicTestnet, block, minerAddress)
}
```

Move the existing body into `ProcessBlockFor` and replace the final reward call with:

```go
totalReward, err := CreditMinerRewardFor(
	workingState,
	policy,
	block.Height,
	minerAddress,
	totalMinerFees,
)
```

Validate `policy` at the beginning of `ProcessBlockFor` before cloning state.

- [ ] **Step 4: Run all blockchain tests**

```bash
go test ./blockchain -count=1
```

Expected: PASS; existing public-testnet block tests remain unchanged.

- [ ] **Step 5: Commit**

```bash
git add blockchain/block_processor.go blockchain/block_processor_test.go
git commit -m "feat(blockchain): thread monetary policy through block processing"
```

---

### Task 6: Add exhaustive consensus invariants and regression verification

**Files:**
- Modify: `consensus/rewards_test.go`
- Modify: `blockchain/rewards_test.go`
- Modify: `blockchain/block_processor_test.go`

**Interfaces:**
- No new production interface; this task closes the verification gates from the spec.

- [ ] **Step 1: Add exhaustive per-epoch remainder test**

For every epoch, prove exactly `remainder` blocks receive `base+1` and all others receive `base`:

```go
func TestMainnetRemainderDistributionEveryEpoch(t *testing.T) {
	for epochIndex, epoch := range params.MainnetEmissionEpochs {
		base := epoch.Issuance / params.MainnetEpochLength
		remainder := epoch.Issuance % params.MainnetEpochLength
		var sum uint64
		var plusOne uint64
		for offset := uint64(0); offset < params.MainnetEpochLength; offset++ {
			height := uint64(epochIndex)*params.MainnetEpochLength + offset + 1
			got, err := BlockSubsidyFor(params.MonetaryPolicyMainnet, height)
			if err != nil { t.Fatal(err) }
			if got == base+1 { plusOne++ } else if got != base {
				t.Fatalf("epoch %d offset %d unexpected reward %d", epochIndex+1, offset, got)
			}
			sum += got
		}
		if plusOne != remainder {
			t.Fatalf("epoch %d expected %d plus-one blocks got %d", epochIndex+1, remainder, plusOne)
		}
		if sum != epoch.Issuance {
			t.Fatalf("epoch %d expected issuance %d got %d", epochIndex+1, epoch.Issuance, sum)
		}
	}
}
```

- [ ] **Step 2: Add final-height and post-cap reward-credit tests**

Prove mainnet height `5_259_600` receives its deterministic last subsidy, height `5_259_601` receives zero subsidy, and miner fees still credit the miner without increasing `IssuedSupply` after subsidy ends.

Use:

```go
fees := uint64(9_000)
got, err := CreditMinerRewardFor(state, params.MonetaryPolicyMainnet, params.MainnetFinalSubsidyHeight+1, "miner", fees)
if err != nil { t.Fatal(err) }
if got != fees { t.Fatalf("expected fee-only reward %d got %d", fees, got) }
if state.IssuedSupply() != before { t.Fatalf("fee-only block minted new supply") }
```

- [ ] **Step 3: Add unknown-policy atomicity test**

Call `ProcessBlockFor` with policy `255` and assert an error plus unchanged state snapshot/issued supply. This proves invalid configuration cannot partially mutate chain state.

- [ ] **Step 4: Run package and repository-wide verification**

```bash
go test ./params ./consensus ./blockchain -count=1
go test ./... -count=1
go vet ./...
```

Expected: all commands PASS.

- [ ] **Step 5: Commit**

```bash
git add consensus/rewards_test.go blockchain/rewards_test.go blockchain/block_processor_test.go
git commit -m "test(consensus): prove 51M mainnet emission invariants"
```

---

### Task 7: CI review and implementation handoff only — no deployment

**Files:**
- No production file changes required unless CI exposes a real regression.

**Interfaces:**
- Produces a verified implementation branch ready for code review, not deployment.

- [ ] **Step 1: Push the implementation branch and inspect GitHub Actions**

Run the repository's normal CI from the implementation branch. Do not add a deployment workflow and do not touch Seed-1/Seed-2.

- [ ] **Step 2: Verify checks**

Required evidence:

```text
- go test ./... => PASS
- go vet ./... => PASS
- existing public-testnet reward tests => PASS
- 40-epoch mainnet boundary tests => PASS
- exact 5,100,000,000,000,000 base-unit cumulative issuance => PASS
- height 5,259,601 subsidy => 0
```

- [ ] **Step 3: Review diff specifically for accidental testnet changes**

Confirm no deployment config, faucet code, wallet code, treasury address, transaction-fee percentages, genesis data, or live node configuration changed.

- [ ] **Step 4: Stop before deployment/activation**

Report implementation commit(s), CI run IDs, and remaining mainnet activation work. Mainnet genesis, network identity, activation wiring, and deployment remain separate explicit work and must not be inferred from this implementation.
