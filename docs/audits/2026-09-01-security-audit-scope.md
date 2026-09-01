# Security audit scope — file map and review questions

**Target commit:** `a63fe384a75a8857f02a78b612be5b9b0a233cb8`  
**Prepared for:** Independent external auditor  
**Owner:** Kk

---

## 1. Monetary policy & supply cap

**Question:** Can mainnet mint beyond 51,000,000 SUDH under any code path?

| Path | Files |
| --- | --- |
| Emission table | `params/mainnet_emission.go`, `params/monetary.go` |
| Subsidy calculation | `consensus/rewards.go` |
| Block rewards | `blockchain/rewards.go`, `blockchain/block_processor.go` |
| Policy-bound minting | `blockchain/state.go` (`NewStateFor`, `MintSupplyFor`, `EnsureMonetaryPolicy`) |
| Tests | `consensus/rewards_test.go`, `blockchain/supply_test.go`, `blockchain/mainnet_monetary_rehearsal_test.go` |

**Scripts:** `bash ./scripts/mainnet-monetary-rehearsal.sh`

---

## 2. Network identity & genesis isolation

**Question:** Can a node accidentally join wrong network or accept wrong genesis?

| Path | Files |
| --- | --- |
| Network IDs | `params/network.go` |
| Chain constructors | `blockchain/chain.go` (`NewChainFor`, `ValidateChainGenesis`) |
| Mainnet genesis candidate | `blockchain/genesis_mainnet.go` |
| Node startup | `cmd/sudharmad/main.go` |
| P2P handshake | `p2p/network.go` (`SetLocalNetworkID`) |
| Launch gates | `params/readiness.go`, `cmd/sudharma-mainnet-readiness/` |

**Verify:** `sudharmad -network mainnet` must fail until activation.

---

## 3. Block, transaction, reorg

**Question:** Can invalid blocks/transactions alter confirmed state or exceed supply?

| Path | Files |
| --- | --- |
| Block processing | `blockchain/block_processor.go`, `blockchain/reorg.go` |
| Transactions | `blockchain/transaction_processor.go`, `transactions/` |
| PoW validation | `pow/pow.go`, `consensus/` |
| Adversarial tests | `blockchain/adversarial_*.go` |

---

## 4. P2P sync & mempool

**Question:** Can a peer cause unauthorized state transitions or resource exhaustion?

| Path | Files |
| --- | --- |
| P2P core | `p2p/*.go` |
| Mempool | `blockchain/mempool/` |
| Limits | `p2p/` transport limits, Step 57/58 docs |

---

## 5. Wallet & key custody

**Question:** Can public surfaces exfiltrate seeds or sign unintended transactions?

| Path | Files |
| --- | --- |
| CLI wallet | `cmd/sudharma-wallet/`, `wallet/` |
| Android wallet | `android/` (if in scope for mobile audit) |
| Website | `web/` — faucet, downloads, no seed prompts |

**Boundary:** RPC and website must never accept private keys.

---

## 6. HTTP RPC & public proxy

**Question:** Are public endpoints appropriately bounded?

| Path | Files |
| --- | --- |
| Node RPC server | `rpc/server.go`, `rpc/*.go` |
| GPU mining API | `rpc/gpu_mining.go`, `rpc/mining_compat.go` |
| Public Lambda proxy | `deployment/testnet/public-rpc/lambda/index.mjs` |
| API docs | `docs/rpc.md` |
| Contract tests | `scripts/check-mining-api-contract_test.sh`, `scripts/check-explorer-api-contract_test.sh` |

**Live testnet base URL (read-only smoke):** `https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com`

---

## 7. GPU mining & pool stack

**Question:** Can non-GPU work, invalid shares, or pool logic steal rewards?

| Path | Files |
| --- | --- |
| Mining policy | `params/mining.go`, `params/mining_readiness.go` |
| Miner client | `gpuminer/`, `cmd/sudharma-miner/` |
| Pool engine | `pool/engine.go`, `pool/share.go`, `pool/payout.go` |
| Stratum | `pool/stratum/`, `gpuminer/stratum/` |
| Architecture | `docs/audits/2026-08-31-mainnet-gpu-mining-architecture.md`, `docs/audits/2026-08-31-pool-mining-architecture.md` |

**Scripts:** `bash ./scripts/pool-mining-smoke_test.sh`, `bash ./scripts/probe-testnet-mining-rpc.sh`

---

## 8. Operator & deployment safety

**Question:** Can CI/workflows accidentally activate mainnet or leak secrets?

| Path | Files |
| --- | --- |
| Workflow gates | `.github/workflows/` (manual `workflow_dispatch`, `confirm=PUBLISH`) |
| Secret scanning | `scripts/check-tracked-secrets.sh`, `scripts/check-tracked-secrets_test.sh` |
| Mainnet deploy scaffold | `deployment/mainnet/` |
| Testnet operator | `deployment/testnet/` |
| Live workflow safety tests | `scripts/live-workflow-trigger-safety.test.mjs` |

---

## 9. Readiness & activation gates

**Question:** Is it possible to bypass human launch gates in code?

| Check | Files |
| --- | --- |
| `MainnetLaunchAuthorized = false` | `params/network.go` — must not be true on audited commit |
| `MainnetMiningAuthorized = false` | `params/mining.go` |
| `MainnetGenesisTimestamp = 0` | `params/network.go` |
| Readiness reporting | `params/readiness.go`, `cmd/sudharma-mainnet-readiness/` |

**Contract:** `bash ./scripts/check-mainnet-readiness-contract_test.sh`

---

## Severity guidance for report

| Severity | Example |
| --- | --- |
| Critical | Mainnet mint bypass, remote key exfiltration, unauthorized block acceptance |
| High | Pool payout manipulation, RPC auth bypass on privileged routes |
| Medium | DoS without state corruption, missing rate limits |
| Low | Hardening, logging gaps |
| Informational | Documentation drift |

---

## Post-audit evidence

Owner records outcome in private vault using `docs/audits/2026-08-31-security-audit-evidence-template.md`. Gate in `params/readiness.go` updates only after owner review of signed report.
