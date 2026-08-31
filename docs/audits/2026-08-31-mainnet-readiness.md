# Audit: Mainnet readiness freeze (not a launch)

**Date:** 2026-08-31  
**Branch:** `cursor/mainnet-readiness-8441`  
**Priority:** Mainnet readiness first. GPU mining (Windows/HiveOS) and DEX/CEX listing stay deferred until this freeze is reviewed.

## What this stage does

Closes the remaining **engineering** gaps called out after Mainnet Tokenomics v1, without activating mainnet or touching live testnet:

- Policy-aware `MintSupplyFor` so mainnet cannot mint past **51,000,000 SUDH** even though testnet still uses a 51B ceiling
- Isolated mainnet P2P identity `sudharma-mainnet-1` (handshake still `sudharma-testnet-1`)
- Isolated mainnet genesis **candidate** (`Sudharma Network Mainnet Genesis Block v1`); default `NewChain()` still uses public-testnet genesis
- `params.MainnetLaunchAuthorized = false` and `sudharmad -network mainnet` exits
- Explicit readiness gates that fail closed until audit, timestamp freeze, seeds, and a human launch decision

## What this stage does not do

- Does not launch mainnet or change public-testnet genesis, rewards, or P2P ID
- Does not publish mainnet seeds or arm GPU-PoW
- Does not list SUDH on a DEX or CEX
- Does not claim the project is mainnet-ready

## Remaining human / operational gates

| Gate | Status |
| --- | --- |
| Merge/review Mainnet Tokenomics v1 (PR #76) | Open for review |
| Merge/review Mainnet readiness freeze (PR #77) | Open for review |
| Independent production security audit | Not started |
| Freeze mainnet genesis unix timestamp (replace `0`) | Not frozen |
| Flip `MainnetLaunchAuthorized` in a dedicated activation PR | Forbidden here |
| Publish mainnet seed topology and operator runbook | Draft scaffold at `deployment/mainnet/` + `docs/audits/2026-08-31-mainnet-launch-operator-runbook.md` |
| DEX wrapped-asset / CEX integration pack | Deferred (native coin; Uniswap-style DEX needs a wrap/bridge after launch) |
| GPU Windows + HiveOS miner packages | Testnet **live** (PR #79 merged; `/v1/mining/work` returns candidate blocks). Mainnet uses same API shape; design at `docs/audits/2026-08-31-mainnet-gpu-mining-architecture.md` |

## Operator check

```bash
go test ./params ./consensus ./blockchain ./p2p ./cmd/sudharmad -count=1
go run ./cmd/sudharma-mainnet-readiness
go run ./cmd/sudharma-mainnet-genesis-info
bash ./scripts/mainnet-monetary-rehearsal.sh
bash ./scripts/check-mainnet-readiness-contract_test.sh
bash ./scripts/check-mainnet-go-live-readiness_test.sh
```
