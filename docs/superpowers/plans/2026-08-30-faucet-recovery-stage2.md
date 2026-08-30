# Faucet Recovery Stage 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore a testable, promotable public-testnet faucet slice on top of the canonical project without importing unrelated diverged work or deploying automatically.

**Architecture:** Start from `integration/canonical-project` and copy only the hardened public-RPC/faucet runtime slice that is required for faucet operation. The public RPC Lambda is shared with website visitor counting and explorer reads, so recovery must preserve those capabilities too. A dedicated deployment workflow builds and tests the artifact on branch pushes, but AWS mutation is manual-only; the exact artifact produced by the successful build job is the artifact eligible for promotion. Deployment forces the faucet disabled before installing code, verifies shared public routes, then enables the faucet under fail-closed readiness checks.

**Tech Stack:** GitHub Actions, Node.js 24, AWS Lambda/API Gateway, DynamoDB, AWS Secrets Manager, Sudharma RPC.

**Spec:** `docs/superpowers/specs/2026-08-26-public-testnet-wallet-faucet-design.md` from the historical faucet branch; Stage 2 intentionally implements only the server-side public-RPC/faucet recovery slice plus the shared-Lambda compatibility required to avoid regressions.

## Global Constraints

- Base every Stage 2 change on canonical commit `8efb33ef03f273511fc8264b6092de3f845665c7`.
- Do not merge to `integration/canonical-project` or `main` without explicit approval.
- Do not deploy to AWS from a branch push.
- Never store a faucet private key in the repository or GitHub Actions inputs; load it from AWS Secrets Manager at runtime.
- Persist grant/claim/idempotency state in DynamoDB.
- Preserve normal public RPC availability if faucet activation fails.
- Preserve existing Lambda environment variables when toggling faucet state.
- Preserve shared `/v1/website/visitors` and `/v1/explorer/*` behavior.
- A tested artifact and a deployed artifact must come from the same workflow run.

---

### Task 1: Deployment contract regression test

**Files:**
- Create: `scripts/check-faucet-deploy-contract_test.sh`
- Create: `.github/workflows/faucet-recovery-ci.yml`

- [x] Require manual-only `workflow_dispatch` deployment.
- [x] Require same-run artifact download.
- [x] Require faucet disabled before code installation.
- [x] Require preserved Lambda environment variables.
- [x] Require `/v1/faucet/info` and deep `/v1/faucet/health` readiness checks.
- [x] Require unexpected-error rollback after activation.
- [x] Require visitor runtime packaging, visitor route preservation, explorer route/query preservation, and shared-route smoke checks.

### Task 2: Restore the hardened faucet Lambda slice

**Files:**
- `deployment/testnet/public-rpc/lambda/package.json`
- `router.mjs`, `router.test.mjs`
- `upstream.mjs`, `upstream.test.mjs`
- `faucet.mjs`, `faucet.test.mjs`
- `faucet-runtime.mjs`, `faucet-runtime.test.mjs`
- `index.mjs`, `index.test.mjs`
- `visitor-runtime.mjs`, `visitor-runtime.test.mjs`
- `shared-routes-regression.test.mjs`

- [x] Restore hardened faucet runtime selectively; no wholesale diverged branch merge.
- [x] Keep `/v1/faucet/health` local to the faucet runtime.
- [x] Preserve website visitor read/write handling locally.
- [x] Preserve read-only explorer routing, validated query forwarding, and browser CORS.
- [x] Run the full Node test suite and tracked-secret safety tests in CI.

### Task 3: Make artifact promotion manual, same-run, and fail-closed

- [x] Build/test/package on branch push.
- [x] Gate AWS mutation behind `workflow_dispatch` + `deploy: true`.
- [x] Snapshot/merge Lambda environment instead of replacing it.
- [x] Force `FAUCET_ENABLED=false` before installing new Lambda code.
- [x] Smoke-test `/v1/status`, `/v1/website/visitors`, and `/v1/explorer/status` before faucet activation.
- [x] Enable under an `ERR` rollback trap.
- [x] Require valid `/v1/faucet/info` and `/v1/faucet/health` with `ready: true`.
- [x] Recheck shared public endpoints after activation.
- [x] Roll back prior Lambda code and original environment if either pre-activation or post-activation shared-route checks fail.
- [x] Prove `lambda:GetFunction` rollback permission in the read-only AWS preflight without exposing `Code.Location`.

### Task 4: Verification checkpoint

Latest verified deployable source commit:

`caef6f1f77138891e864191f0b08cd4d898426a7`

Evidence:
- Post-activation rollback RED commit `2d4e1506269868b0f1d41f67cdec8be26e9339d6`; Testnet Public RPC run `33307921758` failed at deployment-contract verification before the rollback existed.
- Testnet Public RPC run `33307984250`: deployment contract, secret-safety, Lambda tests, package and artifact upload all passed; `aws-preflight` and `deploy` correctly skipped on push.
- Faucet Recovery CI run `33307984249`: deployment-contract and Lambda test jobs passed.
- Artifact `sudharma-testnet-wallet-proxy`, ID `9731081523`.
- Artifact SHA-256: `28d09f0a502ad250b99a741245887d45b103458079d9b4b305c2f69810337f6f`.

### Remaining live boundary

A manual read-only preflight run remains required with `preflight: true`, `deploy: false`. It proves GitHub OIDC trust and required AWS read permissions for `feature/faucet-recovery-stage2` without changing AWS. If it succeeds, a later separately approved `deploy: true` run may perform the guarded rollout. Do not merge the draft PR until live activation evidence is captured and explicit approval is given.
