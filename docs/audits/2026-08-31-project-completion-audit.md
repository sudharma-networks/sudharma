# Sudharma project completion audit

**Recorded:** 2026-08-31  
**Branch:** `cursor/canonical-integration-guard-8441`  
**PR:** [#69](https://github.com/sudharma-networks/sudharma/pull/69)  
**Canonical `main`:** `633843d3719645cf7e81c51ff47cfaad5374c4c7`

This audit is a source-control + read-only live smoke snapshot. It is not a
deployment attestation and does not authorize AWS changes, consensus activation,
or mainnet launch.

## 1. What the project is

Sudharma Network is an open-source pre-mainnet Proof-of-Work blockchain (native
coin **SUDH**) with node, wallet, P2P, mining, public testnet RPC/faucet/explorer
surfaces, and a public website under active development. Consensus rules, APIs,
wallet formats, and network parameters may still change before mainnet.

## 2. Completed stages 0–6

| Stage | Scope | Status |
| --- | --- | --- |
| **0** | Canonical wallet / faucet / demand-miner spine + manual-only safety gate | Complete |
| **1** | Faucet recovery reconciliation | Complete (`66b8cbd`) |
| **2** | Explorer API + website foundation | Complete (`9188d01`) |
| **3** | RC verification at one commit | Complete (`5f92589`) |
| **4** | Operator go-live toolkit (runbook, evidence, verifier) | Complete (`04a7c12`) |
| **5** | Operator-gated core go-live | Core complete; website publish + Android APK deferred |
| **6** | Public surface hardening (faucet web + CORS + honest status) | Code complete; Lambda CORS needs operator redeploy |
| **7** | Public surface operator closure (CORS verify + Amplify promote) | **Complete** — CORS live, website published |

### Stage 5 operator outcomes

| Step | Workflow | Outcome |
| --- | --- | --- |
| 1 | Explorer Seed RPC Deploy | Both seeds upgraded (`sudharma.service`) |
| 2 | Testnet Public RPC | Lambda deploy with rollback protection |
| 3 | Demand miner auto-deploy | Success, often skipped when no chain work |
| 4 | provision-website-visitor-counter | Provisioned + endpoint committed |
| 5 | Website publish | **Deferred by operator** |
| 6 | Faucet Enable Public | Enabled |
| 7 | Android APK release | **Deferred by operator** |

### Stage 6 code deliverables

- `web/lib/faucet-config.ts`, `web/lib/faucet-api.ts`
- `web/components/faucet-panel.tsx`, `/faucet` page
- Lambda faucet CORS + `OPTIONS` in `deployment/testnet/public-rpc/lambda/index.mjs`
- Honest status on home / roadmap / testnet pages
- Docs: `docs/rpc.md`, `docs/testnet-android.md`, Stage 6 audit note

## 3. Live infrastructure (smoke 2026-08-31)

| Surface | Endpoint | Status |
| --- | --- | --- |
| Public RPC | `https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com` | Live — height **~467+**, `network: sudharma`, peers 2 |
| Explorer API | `…/v1/explorer/*` | Live — sources `seed-1`, `seed-2`, `mempool`, `demand-miner`; CORS present |
| Faucet API | `…/v1/faucet/info`, `/health` | Live — `enabled: true`, `ready: true` |
| Faucet CORS | browser `Access-Control-*` on faucet | Live after redeploy — `verify-faucet-browser-cors.mjs` passes |
| Visitor counter | `https://b8dr97u4ob.execute-api.ap-south-1.amazonaws.com` | Live |
| Amplify website | `https://feature-website-foundation.d2mqyt0bt8sl9s.amplifyapp.com/` | Live — serves Stage 6 faucet UI and approved tokenomics copy |

## 4. Deferred / incomplete operator items

1. ~~Redeploy public RPC Lambda for faucet CORS~~ — **done** (run `33410467973`); verified `PASS`.
2. ~~Website publish to Amplify~~ — **done** (`dbeec63`); live `/faucet` serves the request UI.
3. Android APK release (Stage 5 step 7, still deferred).
4. Optional private deployment evidence file verify (`assemble` + `verify` scripts).
5. Demand-miner binary promotion when/if chain work requires it.

## 5. Independent review lines (not mixed in)

| Line | Status |
| --- | --- |
| GPU / Khushi (`feature/gpu-pow-v1`) | Isolated; do not activate via testnet go-live |
| Mainnet Tokenomics v1 (`feature/mainnet-tokenomics-v1`) | Separate consensus review; inventory notes failing rewards test |

## 6. Pending to project / mainnet completion

### Operator near-term (public surface coherence)

1. Green CI on Stage 6 commit; review/merge PR #69 when ready.
2. Redeploy Testnet Public RPC Lambda (CORS).
3. Publish website static build with faucet + status honesty.
4. Publish Android APK when stability criteria are met.
5. Optional: private evidence vault verify against RC commit.

### Product hardening

- Android wallet hardening and downloads provenance.
- Explorer polish beyond v1 read-only surfaces.
- Demand-miner operational hardening.
- GPU mining remains experimental / separate review.

### Pre-mainnet blockers

- Independent production security audit (none yet).
- Freeze consensus-critical parameters / genesis for mainnet.
- Complete Mainnet Tokenomics v1 review line.
- Explicit mainnet launch decision after operational readiness.

## 7. PR #69 readiness

- Large integration PR; draft historically used for staged landing.
- Core Stages 0–5 + Stage 6 code intended on this branch.
- Merge only after Stage 6 CI is green and reviewers accept deferred publish items.
- Live Amplify and faucet CORS remain operator follow-ups after merge or before declaring “public web done.”

## 8. Explicit non-goals (still isolated)

- GPU / Khushi consensus activation
- Mainnet Tokenomics / mainnet launch
- Automated (non–`workflow_dispatch`) AWS mutation
- Committing private evidence digests into git
- Website challenge-round faucet UI (Android wallet only)
- Claiming monetary value for Test SUDH

## Bottom line

**Public testnet core is live** (seeds, RPC, explorer API, faucet enable, visitor
counter). Stages **0–6 code** on this branch close the canonical integration and
public faucet web path. Remaining work to a coherent public surface is mostly
**operator publish/redeploy**; remaining work to **mainnet** is security review,
tokenomics, genesis freeze, and an explicit launch decision.
