# PR #77 reviewer summary — Mainnet readiness freeze

**PR:** [#77](https://github.com/sudharma-networks/sudharma/pull/77)  
**Base:** `cursor/mainnet-tokenomics-v1-8441` (merge **after** #76)  
**Scope:** Mainnet engineering freeze + testnet mining stack. **Does not activate mainnet.**

## What to verify (10 minutes)

1. **Launch gates fail closed:** `go run ./cmd/sudharma-mainnet-readiness` → `launch_ready: false`
2. **Mining stack ready on testnet:** `mining_stack_ready: true`, `mainnet_mining_authorized: false`
3. **Isolated mainnet identity:** `sudharma-mainnet-1` distinct from testnet; `sudharmad -network mainnet` blocked
4. **Policy-bound minting:** `NewStateFor` + cross-policy mint rejection (51M cap cannot bypass via testnet wrapper)
5. **No forbidden flags:** grep confirms no `MainnetLaunchAuthorized = true` in params/cmd/blockchain

## Key deliverables

| Area | Location |
| --- | --- |
| Readiness gates | `params/readiness.go`, `params/mining_readiness.go` |
| Policy-bound state | `blockchain/state.go` (`NewStateFor`, `EnsureMonetaryPolicy`) |
| Mainnet genesis candidate | `blockchain/genesis_mainnet.go`, `cmd/sudharma-mainnet-genesis-info` |
| GPU solo API | `rpc/gpu_mining.go` |
| Pool stack | `pool/`, `cmd/sudharma-pool`, `gpuminer/stratum/` |
| Operator scaffolds | `deployment/mainnet/`, merge review + genesis freeze templates |

## One-command verification

```bash
bash ./scripts/verify-mainnet-merge-readiness.sh pr77
```

## Expected JSON (before launch)

```json
{
  "launch_authorized": false,
  "launch_ready": false,
  "mining_stack_ready": true
}
```

Mining readiness CLI: `stack_ready: true`, `mainnet_mining_authorized: false`.

## Merge blocker checklist

- [ ] PR #76 merged first (or base branch contains #76)
- [ ] CI green on #77
- [ ] `bash ./scripts/check-mainnet-merge-review-contract_test.sh` passes
- [ ] No `MainnetLaunchAuthorized = true` / `MainnetMiningAuthorized = true` / non-zero genesis timestamp

## Explicit non-goals in this PR

- Mainnet seed deploy
- Genesis timestamp freeze
- Security audit attestation
- Windows miner GitHub release (operator parallel path)
- Public testnet pool host deploy (operator parallel path)

## After merge

1. Record merge commit for operator evidence
2. Security audit (`docs/audits/2026-08-31-security-audit-evidence-template.md`)
3. Genesis freeze PR (`docs/audits/2026-08-31-mainnet-genesis-freeze-template.md`)
4. Seed topology review (`deployment/mainnet/OPERATOR-CHECKLIST.md`)
