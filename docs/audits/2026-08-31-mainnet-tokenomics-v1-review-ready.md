# Audit: Mainnet Tokenomics v1 — Implementation Review Ready

**Date:** 2026-08-31  
**Branch:** `cursor/mainnet-tokenomics-v1-8441`  
**Scope:** Stage 8 — encode and verify mainnet monetary policy in consensus/blockchain code paths without activating mainnet.

## What shipped

- `params.MonetaryPolicy` with public-testnet and mainnet IDs
- Exact 40-epoch mainnet emission table totaling **51,000,000 SUDH** (5.1e15 base units)
- `consensus.BlockSubsidyFor(policy, height)` with public-testnet wrapper `BlockSubsidy`
- `blockchain.CreditMinerRewardFor` / `ProcessBlockFor` with public-testnet wrappers unchanged
- Invariant tests: per-epoch remainder distribution, cumulative hard cap, final height / post-cap fee-only, unknown-policy atomicity

## Verification

```text
go test ./... -count=1  => PASS
go vet ./...            => PASS
```

Public-testnet reward behavior preserved via compatibility wrappers (`ProcessBlock`, `CreditMinerReward`, `BlockSubsidy`).

## Explicit non-goals (not done)

- No mainnet genesis, network ID, or activation wiring
- No live testnet reward or deployment changes
- No faucet / website / wallet / treasury changes
- No claim of mainnet readiness on public surfaces

## Review note

`State.MintSupply` still enforces the public-testnet `params.MaxSupply` ceiling. Mainnet safety for this stage relies on `stateRemainingSupplyFor` clipping against `params.MaxSupplyFor(policy)` before mint. A later activation PR should make `MintSupply` policy-aware (or remove the hardcoded testnet ceiling) before mainnet genesis.

## Remaining before mainnet launch

- Human review of this implementation
- Policy-aware mint ceiling (or equivalent) if required for activation
- Security audit, genesis freeze, tokenomics activation decision, launch decision
