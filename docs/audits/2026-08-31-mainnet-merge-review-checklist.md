# Mainnet merge review checklist

**Date:** 2026-08-31  
**Purpose:** Human review path for landing mainnet engineering without activating mainnet.

## Merge order

| Step | PR | Base | Review focus |
| --- | --- | --- | --- |
| 1 | [#76](https://github.com/sudharma-networks/sudharma/pull/76) Mainnet Tokenomics v1 | `main` | 51M cap, 40-epoch table, `BlockSubsidyFor`, public-testnet wrappers unchanged |
| 2 | [#77](https://github.com/sudharma-networks/sudharma/pull/77) Mainnet readiness freeze | `cursor/mainnet-tokenomics-v1-8441` | Isolated genesis, launch gates, mining stack docs, policy-bound state minting |

Do **not** merge #77 before #76. Rebase #77 onto merged #76 if needed.

## CI gates (both PRs)

- [ ] `test` job green
- [ ] `deployment-contract` green
- [ ] `demand-miner` green
- [ ] `lambda-tests` green

## Local verification before approving #76

```bash
go test ./... -count=1
go vet ./...
go test ./consensus -run TestMainnet -count=1
go test ./blockchain -run 'TestProcessBlockForMainnet|TestMintSupplyForMainnet' -count=1
```

Confirm public-testnet behavior is unchanged:

- `BlockSubsidy(0)` still returns genesis subsidy on testnet paths
- Live testnet genesis / P2P ID / reward schedule untouched in params

## Local verification before approving #77

```bash
go test ./... -count=1
go run ./cmd/sudharma-mainnet-readiness | jq .
go run ./cmd/sudharma-mining-readiness | jq .
go run ./cmd/sudharma-mainnet-genesis-info | jq .
bash ./scripts/mainnet-monetary-rehearsal.sh
bash ./scripts/check-mainnet-readiness-contract_test.sh
bash ./scripts/check-mainnet-go-live-readiness_test.sh
bash ./scripts/check-pool-mining-contract_test.sh
bash ./scripts/check-mining-readiness-contract_test.sh
bash ./scripts/pool-mining-smoke_test.sh
```

Expected readiness output:

- `launch_authorized: false`
- `launch_ready: false`
- `mining_stack_ready: true`
- `mainnet_mining_authorized: false` (mining-readiness CLI)

Forbidden in these PRs:

- `MainnetLaunchAuthorized = true`
- `MainnetMiningAuthorized = true`
- `MainnetGenesisTimestamp != 0`
- Live testnet genesis / AWS auto-deploy changes

## After merge to `main`

1. Tag or record the merge commit for operator evidence
2. Proceed to **security audit** and **genesis timestamp freeze** (human gates)
3. Review mainnet seed topology draft (`deployment/mainnet/OPERATOR-CHECKLIST.md`)
4. Open a **dedicated activation PR** only after all human gates close

## Parallel (does not block merge)

- Republish Windows GPU miner: `.github/workflows/windows-gpu-miner-publish.yml` with `confirm=PUBLISH`
- Optional public testnet pool deploy: `deployment/testnet/pool-operator-runbook.md`
