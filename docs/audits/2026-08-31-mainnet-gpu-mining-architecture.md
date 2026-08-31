# Mainnet GPU mining architecture (Sudharma / Khushi)

**Date:** 2026-08-31  
**Status:** Engineering design for post-launch mainnet. Public-testnet GPU mining uses the same HTTP work/submit shape; mainnet stays closed until `MainnetMiningAuthorized` and `MainnetLaunchAuthorized`.

## How established PoW chains do it

| Chain | Work delivery | Submission | Reward destination | Notes |
| --- | --- | --- | --- | --- |
| **Bitcoin** | `getblocktemplate` / `getwork` via node RPC | `submitblock` | Coinbase pays miner-chosen address | Pools use Stratum on top of node work |
| **Ethereum (pre-merge PoW)** | `eth_getWork` JSON-RPC | `eth_submitWork` | Coinbase in block header | Work package: header hash, seed, target |
| **Ravencoin (RVN)** | `getblocktemplate` + KawPoW | `submitblock` | Coinbase to miner address | GPU-focused; pools use Stratum v1 |
| **Monero** | `getblocktemplate` | `submitblock` | Miner payout in coinbase | RandomX is CPU-first; architecture still applies |

Common patterns:

1. **Node issues candidate blocks** — not raw hash jobs disconnected from consensus.
2. **Miner chooses payout address** — embedded in coinbase / block reward field before hashing.
3. **Separate read and write paths** — explorers/wallets use read-only APIs; mining uses explicit submit endpoints.
4. **Proxy allowlists** — public Internet sees only mining + wallet routes, not admin RPC.
5. **Failover seeds** — wallets and miners try seed-1 then seed-2 (Sudharma public-testnet Lambda already does this).
6. **Compatibility aliases** — work JSON includes `pow_compat.getblocktemplate` (RVN/BTC field names) and `pow_compat.eth_getWork` (pre-merge Ethereum PoW names) so pool software can integrate without legacy JSON-RPC on sudharma-rpcd.

## Sudharma public-testnet (live path)

Already implemented on `main` after PR #79:

```
Windows GPU miner  →  public HTTPS proxy  →  seed-1 / seed-2 nginx (29100)
                     /v1/mining/work          sudharma-rpcd /v1/mining/work
                     /v1/mining/submit        sudharma-rpcd /v1/mining/submit
```

- **Algorithm:** `sudharma-gpupow-v1` (Khushi). CPU/ASIC rejected.
- **Work:** Full candidate block with `reward_address` set to the miner's wallet.
- **Submit:** Solved block JSON; node runs `AcceptBlock` + `BroadcastBlock`.
- **Parallel demand miner:** Separate process; first accepted block wins. Demand miner unchanged.

Testnet operator configs: `deployment/testnet/gpu-miner*.example.json`.

## Sudharma mainnet (planned — not live)

Mainnet reuses the **same miner client and RPC shape** but different gates and economics:

| Concern | Mainnet behavior |
| --- | --- |
| Authorization | `params.MainnetMiningAuthorized = false` until launch PR |
| Network identity | `sudharma-mainnet-1` P2P ID (isolated from testnet) |
| Emission | 40-epoch table; hard cap **51,000,000 SUDH** |
| Post-cap mining | Fee-only blocks after height **5,259,600** |
| Seeds | Dedicated mainnet seed topology (unpublished until launch) |
| Public entry | Mainnet HTTPS RPC proxy with nginx allowlist (mirror testnet) |

### Recommended mainnet miner UX (aligned with RVN/Ethereum solo mining)

1. **One-click / CLI:** paste wallet address once; miner remembers it (already on Windows testnet).
2. **Work polling:** `POST /v1/mining/work` with `{ "address": "<40-hex>" }`.
3. **Local hashing:** Khushi GPU backend (`cuda` / `opencl`) or reference loop for staging.
4. **Submit:** `POST /v1/mining/submit` with solved block JSON.
5. **Failover:** client tries seed-1 → seed-2 → public proxy (already in `gpuminer.FailoverClient`).

### Future optional upgrades (post-v1 launch)

These match what RVN/ETH ecosystems add after solo mining works:

- **Stratum v1 bridge** — for pool operators; translate Stratum shares to Sudharma block submit (out of scope for v1).
- **HiveOS / rig manager pack** — ship Khushi binary + `sudharma-miner --auto` wrapper (Windows one-click already exists).
- **Work versioning** — add `version: 2` Khushi header jobs when GPU-PoW fully replaces SHA256d candidate mining on all networks.
- **Difficulty retarget telemetry** — expose target/difficulty in work JSON (already present) for pool dashboards.

## What must not happen before mainnet launch

- Do **not** set `MainnetLaunchAuthorized = true` or `MainnetMiningAuthorized = true` in autonomous work.
- Do **not** point testnet miners at mainnet endpoints prematurely.
- Do **not** merge testnet demand-miner reward paths into GPU miner APIs.

## Operator verification (mainnet prep)

```bash
go run ./cmd/sudharma-mainnet-genesis-info
go run ./cmd/sudharma-mainnet-readiness
bash ./scripts/mainnet-monetary-rehearsal.sh
bash ./scripts/check-mainnet-readiness-contract_test.sh
```

Public-testnet mining verification:

```bash
curl -fsS -X POST "$RPC/v1/mining/work" \
  -H 'content-type: application/json' \
  --data '{"address":"YOUR_WALLET_ADDRESS"}'
```

## Related docs

- `docs/audits/2026-08-31-mainnet-readiness.md` — engineering freeze gates
- `docs/audits/2026-08-31-mainnet-launch-operator-runbook.md` — human activation sequence
- `deployment/testnet/gpu-miner.example.json` — testnet miner topology (seed-1, seed-2, public proxy)
