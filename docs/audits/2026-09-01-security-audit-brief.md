# Security audit brief — for external reviewers

**Project:** Sudharma Network  
**Audit type:** Independent pre-mainnet security review  
**Target commit:** `a63fe384a75a8857f02a78b612be5b9b0a233cb8` on `main`  
**Date prepared:** 2026-09-01  
**Owner contact:** Kk (project owner)

## Executive summary

Sudharma is an open-source Proof-of-Work blockchain (native coin **SUDH**) with:

- Public testnet live today (seeds, RPC proxy, faucet, explorer, GPU solo mining)
- Mainnet **engineering** merged on `main` but **not activated**
- Human gates still closed: `MainnetLaunchAuthorized = false`, `MainnetMiningAuthorized = false`, genesis timestamp unset

This audit should assess whether the codebase safely supports a future mainnet launch when human gates are cleared. The audit does **not** authorize launch.

## Threat model highlights

| Asset | Risk |
| --- | --- |
| Monetary supply | Mint past 51M cap, subsidy manipulation, fee/subsidy confusion |
| Chain selection | Deep reorg, invalid block acceptance, timestamp abuse |
| Network identity | Testnet/mainnet cross-contamination, wrong genesis |
| Wallet keys | Seed/private key exposure via RPC, website, or logs |
| Public RPC | Unauthorized minting routes, DoS, transaction replay |
| GPU mining | CPU/ASIC bypass, invalid share acceptance, pool payout manipulation |
| Operator surface | Accidental mainnet activation, secret leakage in git |

## In-scope components

1. **Consensus & monetary policy** — 51M cap, 40-epoch emission, policy-bound state minting
2. **Block & transaction processing** — validation, reorg, mempool
3. **P2P** — sync, gossip, network ID handshake
4. **Wallet** — signing, key storage (Android + CLI)
5. **Public HTTP RPC** — node RPC, GPU mining API, public Lambda proxy subset
6. **GPU mining & pools** — work/submit, Stratum, share validation, payout ledgers
7. **Deployment safety** — manual-only workflows, systemd templates, no secrets in repo

## Out of scope (default engagement)

- Live AWS penetration test
- Mainnet production seed operation
- Smart contracts / tokens (not shipped)
- Third-party Khushi GPU binary internals (treat as dependency; review integration boundary only)

## Expected deliverables

1. Written report with severity-classified findings (Critical / High / Medium / Low / Informational)
2. Repro steps and affected commits/files per finding
3. Remediation recommendations
4. Explicit statement: **hold** or **approve for genesis freeze review** (not “launch mainnet now”)

## Verification baseline

Auditors should reproduce tests on the target commit:

```bash
bash ./scripts/verify-mainnet-merge-readiness.sh pr77
bash ./scripts/mainnet-monetary-rehearsal.sh
go test ./... -count=1
bash ./scripts/check-tracked-secrets_test.sh
```

Readiness CLI (must show launch blocked):

```bash
go run ./cmd/sudharma-mainnet-readiness
# launch_authorized: false, launch_ready: false
```

## Key policy constants (verify in code, do not trust this document alone)

- Mainnet max supply: 51,000,000 SUDH
- Mainnet P2P network ID: `sudharma-mainnet-1`
- Public testnet P2P ID: `sudharma-testnet-1`
- Production mining algorithm: `sudharma-gpupow-v1` (GPU-only)
- Launch flags must remain `false` until dedicated activation PR

## Repository

https://github.com/sudharma-networks/sudharma

Detailed file-level scope: `docs/audits/2026-09-01-security-audit-scope.md`
