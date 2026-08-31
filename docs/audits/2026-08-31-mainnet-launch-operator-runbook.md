# Mainnet launch operator runbook (draft)

**Date:** 2026-08-31  
**Status:** Draft checklist only. **Mainnet is not authorized.**

This runbook describes the human/operator steps required after engineering
freezes land (PR #76 tokenomics, PR #77 readiness). It does **not** authorize
launch by itself.

## Preconditions (engineering)

- [ ] Merge PR #76 — Mainnet Tokenomics v1
- [ ] Merge PR #77 — Mainnet readiness freeze
- [ ] `go test ./... -count=1` green on `main`
- [ ] `go run ./cmd/sudharma-mainnet-readiness` shows `launch_ready: false`

## Human gates (required before any activation PR)

| Gate | Owner | Action |
| --- | --- | --- |
| Independent production security audit | External auditor | Record signed audit report in private evidence vault |
| Genesis timestamp freeze | Core team | Choose unix timestamp; update `params.MainnetGenesisTimestamp` in a dedicated PR |
| Mainnet seed topology | Operators | Publish seed addresses, RPC policy, and systemd units (manual deploy only) |
| Launch decision | Project lead | Explicit written approval to set `MainnetLaunchAuthorized = true` |

## Forbidden in readiness PRs

- Do not set `MainnetLaunchAuthorized = true` in the same PR as tokenomics or readiness freeze
- Do not change public-testnet genesis, P2P ID, or live reward schedule
- Do not auto-deploy AWS resources for mainnet
- Do not publish GPU-PoW activation on seeds from this runbook

## Activation sequence (after all gates green)

1. Open a dedicated **activation PR** that only:
   - Sets frozen `MainnetGenesisTimestamp`
   - Sets `MainnetLaunchAuthorized = true` (only after human decision)
   - Documents seed topology and operator verification commands
2. Operator manual deploy of mainnet seeds (`workflow_dispatch` only)
3. Smoke:
   - `sudharmad -network mainnet` must fail until activation PR merges
   - After activation, genesis hash must match published candidate
   - Monetary cap must remain 51,000,000 SUDH
4. Record deployment evidence (private vault; do not commit secrets)

## Verification commands

```bash
go test ./params ./consensus ./blockchain ./p2p ./cmd/sudharmad -count=1
go run ./cmd/sudharma-mainnet-readiness | jq .
bash ./scripts/check-mainnet-readiness-contract_test.sh
```

Expected before launch:

- `launch_authorized: false`
- `launch_ready: false`
- `genesis-timestamp-freeze` gate not ready
- `independent-security-audit` gate not ready

## Deferred until after mainnet launch decision

- GPU Windows/HiveOS public miner packages (stay on `feature/gpu-pow-v1`)
- DEX/CEX listing (native SUDH needs wrap/bridge for Uniswap-style DEXes)
- Public GPU mining RPC on mainnet (requires `MainnetMiningAuthorized`)

## Related PRs

- #76 Mainnet Tokenomics v1
- #77 Mainnet readiness freeze
- #79 One-click GPU miner (public-testnet; separate from demand miner)
