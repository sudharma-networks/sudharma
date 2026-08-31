# Sudharma pool mining architecture

**Date:** 2026-08-31  
**Status:** Reference implementation for public-testnet pool operators. Mainnet pools stay closed until `MainnetMiningAuthorized`.

## Mining modes (industry comparison)

| Mode | How miners get paid | Sudharma support |
| --- | --- | --- |
| **Solo** | Miner keeps full block reward when they find a block | Live via `sudharma-miner` / Windows one-click (`POST /v1/mining/work` + submit) |
| **PPS** (Pay Per Share) | Fixed payout per valid share at pool difficulty | Reference ledger in `pool/payout.go`; operator pays from pool treasury |
| **PPLNS** (Pay Per Last N Shares) | Block reward split across recent shares proportionally | Default scheme in `cmd/sudharma-pool` |
| **FPPS** (Full Pay Per Share) | PPS for shares plus transaction-fee/block handling variant | Supported in payout ledger (`SchemeFPPS`) |
| **Stratum pool** | Miners connect to pool TCP endpoint; pool validates shares and submits blocks | Reference Stratum v1 bridge in `pool/stratum` |

Sudharma nodes still issue **full candidate blocks** (same as solo). The pool server:

1. Fetches work with the **pool payout wallet** embedded in `MinerAddress`.
2. Validates worker nonces at **pool difficulty** (lower than network difficulty).
3. Submits solved blocks to `/v1/mining/submit` when a share meets **network difficulty**.
4. Credits workers through the selected payout scheme.

## Worker identity

Stratum login uses the same pattern as RVN/BTC pools:

```
<40-hex-wallet>.<worker-name>
```

Example: `58a66438f90328d28e443d29a2e55d857732e755.rig1`

Workers never send private keys. The pool maps worker logins to internal balances.

## Stratum v1 contract (Sudharma extension)

Methods:

| Method | Direction | Purpose |
| --- | --- | --- |
| `mining.subscribe` | miner → pool | Session setup; returns extranonce |
| `mining.authorize` | miner → pool | `wallet.worker` login |
| `mining.notify` | pool → miner | Job with candidate block fields + pool target |
| `mining.submit` | miner → pool | `[login, job_id, nonce]` |

`mining.notify` params:

1. `job_id`
2. `height`
3. `parent_hash`
4. `merkle_root`
5. `pool_difficulty`
6. `block_difficulty`
7. `pool_target` (hex)
8. `timestamp`
9. `version`
10. `miner_address` (pool payout wallet for pooled blocks)
11. `clean_jobs` (bool)

## Operator quick start (public-testnet)

```bash
cp deployment/testnet/pool.example.json pool.local.json
# edit payout_address and optional rpc_urls

go run ./cmd/sudharma-pool -config pool.local.json
```

Point GPU miners at `stratum+tcp://YOUR_HOST:3333` with login `YOUR_WALLET.worker`.

CLI flags (without config file):

```bash
go run ./cmd/sudharma-pool \
  -network public-testnet \
  -rpc https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com \
  -payout-address YOUR_WALLET \
  -payout-scheme pplns \
  -pool-difficulty 1 \
  -stratum-listen :3333
```

## Payout scheme selection

| Scheme | Best for | Risk |
| --- | --- | --- |
| **solo** | Pool forwarding solo miners | No share smoothing; miners wait for blocks |
| **pps** | Stable miner income | Pool operator carries variance risk |
| **pplns** | Community pools with loyal miners | Lower operator risk; miners prefer longer sessions |
| **fpps** | Large operators with fee infrastructure | Requires accurate fee accounting |

Default reference pool uses **PPLNS** with a 10,000-share window and 1% fee (`pool_fee_bps: 100`).

## Related docs

- `docs/audits/2026-08-31-mainnet-gpu-mining-architecture.md` — solo mining and node API
- `deployment/testnet/gpu-miner.example.json` — solo miner topology
- `deployment/testnet/pool.example.json` — pool operator config template
