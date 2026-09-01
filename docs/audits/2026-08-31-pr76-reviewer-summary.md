# PR #76 reviewer summary — Mainnet Tokenomics v1

**PR:** [#76](https://github.com/sudharma-networks/sudharma/pull/76)  
**Base:** `main`  
**Scope:** Encode mainnet monetary policy in consensus/blockchain code. **Does not activate mainnet.**

## What to verify (5 minutes)

1. **Hard cap:** `params.MainnetMaxSupply` = 51,000,000 SUDH (5.1e15 base units)
2. **Emission table:** 40 epochs in `params/mainnet_emission.go`; tests sum to cap
3. **Compatibility:** `BlockSubsidy`, `ProcessBlock`, `CreditMinerReward` still use public-testnet policy
4. **No activation:** no `MainnetLaunchAuthorized`, no mainnet genesis timestamp, no seed deploy

## Key files

| File | Purpose |
| --- | --- |
| `params/mainnet_emission.go` | 40-epoch subsidy table |
| `params/monetary.go` | `MonetaryPolicy`, `MaxSupplyFor`, validation |
| `consensus/rewards.go` | `BlockSubsidyFor(policy, height)` |
| `blockchain/rewards.go` | `CreditMinerRewardFor`, `ProcessBlockFor` |
| `consensus/rewards_test.go` | Epoch sums, cap invariants |

## One-command verification

```bash
bash ./scripts/verify-mainnet-merge-readiness.sh pr76
```

## Expected outcomes

- All tests pass
- Live testnet params unchanged (genesis, P2P ID `sudharma-testnet-1`, 51B testnet cap)
- Mainnet policy only reachable through explicit `*For` APIs and tests

## Merge blocker checklist

- [ ] CI green (test, deployment-contract, demand-miner, lambda-tests)
- [ ] No live testnet reward or deployment changes
- [ ] Reviewer agrees 51M cap math matches approved tokenomics doc

## After merge

Rebase PR #77 onto merged `main` (or onto updated `cursor/mainnet-tokenomics-v1-8441` if that branch is kept). See `docs/audits/2026-08-31-mainnet-merge-review-checklist.md`.
