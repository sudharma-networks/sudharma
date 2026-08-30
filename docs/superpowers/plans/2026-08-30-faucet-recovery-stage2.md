# Faucet Recovery Stage 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore a testable, promotable public-testnet faucet slice on top of the canonical project without importing unrelated diverged work or deploying automatically.

**Architecture:** Start from `integration/canonical-project` and copy only the hardened public-RPC/faucet runtime slice required for faucet operation. The public RPC Lambda is shared with website visitor counting and explorer reads, so recovery must preserve those capabilities too. A dedicated deployment workflow builds and tests the artifact on branch pushes, but AWS mutation is manual-only; the exact artifact produced by the successful build job is the artifact eligible for promotion. Deployment snapshots the existing Lambda environment and ZIP, forces the faucet disabled before installing code, verifies shared public routes, rolls back code/environment if pre-activation shared-route checks fail, then enables the faucet under fail-closed readiness checks.

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
- Prove rollback prerequisites in the read-only AWS preflight before allowing deploy.
- Do not expose the presigned Lambda `Code.Location` in preflight output.

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
- [x] Require unexpected-error fail-closed handling after activation.
- [x] Require visitor runtime packaging, visitor route preservation, explorer route/query preservation, and shared-route smoke checks.
- [x] Require prior Lambda ZIP/environment rollback on pre-activation shared-route failure.
- [x] Require read-only preflight proof of `lambda:GetFunction` permission using safe metadata only.

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

### Task 3: Make artifact promotion manual, same-run, rollback-capable, and fail-closed

- [x] Build/test/package on branch push.
- [x] Gate AWS mutation behind `workflow_dispatch` + `deploy: true`.
- [x] Provide separate `workflow_dispatch` read-only AWS/OIDC preflight.
- [x] Verify STS identity, DynamoDB metadata, Secrets Manager metadata, Lambda configuration, and `lambda:GetFunction` in preflight.
- [x] Query only `Configuration.CodeSha256` from preflight `get-function`; do not expose `Code.Location`.
- [x] Snapshot/merge Lambda environment instead of replacing it.
- [x] Snapshot the currently deployed Lambda ZIP before replacement.
- [x] Force `FAUCET_ENABLED=false` before installing new Lambda code.
- [x] Smoke-test `/v1/status`, `/v1/website/visitors`, and `/v1/explorer/status` before faucet activation.
- [x] Restore previous Lambda ZIP and original environment if any pre-activation shared-route check fails.
- [x] Enable under an `ERR` fail-closed trap.
- [x] Require valid `/v1/faucet/info` and `/v1/faucet/health` with `ready: true`.
- [x] Recheck shared public endpoints after activation.

### Task 4: Verification checkpoint

Current verified deployment artifact source commit:

`1860ff0bb61f31ab53fcafb404b8fc1c67215e0b`

TDD evidence:
- Rollback RED commit `28d00285cd149f6fb1ddf0f482ee339c24361733`; Testnet Public RPC run `33306451249` failed deployment-contract verification before shared-Lambda rollback existed.
- Rollback implementation commit `e331e3cb512e7e7e818a80ed7f7eb939fe815650`.
- Rollback contract alignment commit `91736ba4b32aa43310602cd25e8c3b3f92e03fd4`; Testnet Public RPC run `33306543718` and Faucet Recovery CI run `33306543687` passed.
- Rollback-permission preflight RED commit `70c72def9bec58612f7c3a8b32899e960932cf2a`; Testnet Public RPC run `33306930178` failed deployment-contract verification because preflight did not yet prove `lambda:GetFunction` permission.
- Rollback-permission preflight GREEN commit `1860ff0bb61f31ab53fcafb404b8fc1c67215e0b`.
- Testnet Public RPC run `33306964020`: deployment contract, tracked-secret safety, Lambda tests, package and artifact upload all passed; `aws-preflight` and `deploy` correctly skipped on normal push.
- Faucet Recovery CI run `33306964005`: deployment-contract and Lambda-test jobs both passed.

Latest artifact:
- name: `sudharma-testnet-wallet-proxy`
- ID: `9730773422`
- SHA-256: `e19e1e3f5815196f6eb6a7f00b10be42088184ed350b2946d7a1abe7b5bdfa8f`
- size: `2219034` bytes
- created: `2026-08-30T10:38:32Z`
- expires: `2026-09-13T10:38:31Z`

### Remaining live boundary

1. Manually run `.github/workflows/testnet-public-rpc.yml` from `feature/faucet-recovery-stage2` with `preflight: true`, `deploy: false`.
2. Confirm GitHub OIDC can assume `arn:aws:iam::981626123397:role/Sudharma-GitHub-Actions-Testnet` from this branch.
3. Confirm the role can read the faucet DynamoDB table metadata, faucet secret metadata, Lambda configuration, and `lambda:GetFunction` metadata.
4. If preflight succeeds, a later manual run with `deploy: true` can perform the guarded rollout.
5. During rollout, snapshot the current Lambda environment and ZIP before mutation.
6. Force faucet disabled, install exact same-run tested artifact, then run shared-route smoke checks.
7. If shared-route smoke fails, restore previous Lambda code and environment.
8. If shared-route smoke passes, activate faucet and require `/v1/faucet/health` to return `ready: true`.
9. If activation/readiness fails, disable faucet again.
10. Do not exercise payout POST endpoints without explicit approval.

No live AWS deployment, payout, or merge is part of this Stage 2 code verification work.
