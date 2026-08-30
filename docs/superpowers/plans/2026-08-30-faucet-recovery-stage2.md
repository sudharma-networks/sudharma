# Faucet Recovery Stage 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore a testable, promotable public-testnet faucet slice on top of the canonical project without importing unrelated diverged work or deploying automatically.

**Architecture:** Start from `integration/canonical-project` and copy only the hardened public-RPC/faucet runtime slice that is required for faucet operation. A dedicated deployment workflow must build and test the artifact on branch pushes, but AWS mutation is manual-only; the exact artifact produced by the successful build job is the artifact eligible for promotion. Deployment keeps the faucet disabled until normal RPC health succeeds, then enables it and rolls back to disabled if `/v1/faucet/info` fails.

**Tech Stack:** GitHub Actions, Node.js 24, AWS Lambda/API Gateway, DynamoDB, AWS Secrets Manager, Sudharma RPC.

**Spec:** `docs/superpowers/specs/2026-08-26-public-testnet-wallet-faucet-design.md` from the historical faucet branch; Stage 2 intentionally implements only the server-side public-RPC/faucet recovery slice.

## Global Constraints

- Base every Stage 2 change on canonical commit `8efb33ef03f273511fc8264b6092de3f845665c7`.
- Do not merge to `integration/canonical-project` or `main` without explicit approval.
- Do not deploy to AWS from a branch push.
- Never store a faucet private key in the repository or GitHub Actions inputs; load it from AWS Secrets Manager at runtime.
- Persist grant/claim/idempotency state in DynamoDB.
- Preserve normal public RPC availability if faucet activation fails.
- A tested artifact and a deployed artifact must come from the same workflow run.

---

### Task 1: Deployment contract regression test

**Files:**
- Create: `scripts/check-faucet-deploy-contract_test.sh`
- Create: `.github/workflows/faucet-recovery-ci.yml`

**Interfaces:**
- Consumes: repository files only.
- Produces: an executable contract test that fails until the Stage 2 deployment workflow has manual-only deployment semantics.

- [ ] **Step 1: Write the failing test**

The shell test must require `.github/workflows/testnet-public-rpc.yml` to exist and assert all of the following strings are present:

```text
workflow_dispatch:
deploy:
if: github.event_name == 'workflow_dispatch' && inputs.deploy == true
uses: actions/download-artifact@v4
FAUCET_ENABLED=false
/v1/faucet/info
```

It must also fail if the deploy job contains a `push`-only branch equality such as `github.ref == 'refs/heads/feature/public-testnet-wallet-v2'`.

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
bash ./scripts/check-faucet-deploy-contract_test.sh
```

Expected: FAIL because canonical has no Stage 2 public-RPC deployment workflow.

- [ ] **Step 3: Commit the red test**

```bash
git add scripts/check-faucet-deploy-contract_test.sh .github/workflows/faucet-recovery-ci.yml
git commit -m "test(faucet): lock safe deployment contract"
```

### Task 2: Restore the hardened faucet Lambda slice

**Files:**
- Create: `deployment/testnet/public-rpc/lambda/package.json`
- Create: `deployment/testnet/public-rpc/lambda/router.mjs`
- Create: `deployment/testnet/public-rpc/lambda/router.test.mjs`
- Create: `deployment/testnet/public-rpc/lambda/upstream.mjs`
- Create: `deployment/testnet/public-rpc/lambda/upstream.test.mjs`
- Create: `deployment/testnet/public-rpc/lambda/faucet.mjs`
- Create: `deployment/testnet/public-rpc/lambda/faucet.test.mjs`
- Create: `deployment/testnet/public-rpc/lambda/faucet-runtime.mjs`
- Create: `deployment/testnet/public-rpc/lambda/faucet-runtime.test.mjs`
- Create: `deployment/testnet/public-rpc/lambda/index.mjs`
- Create: `deployment/testnet/public-rpc/lambda/index.test.mjs`

**Interfaces:**
- Consumes: Sudharma seed RPC account/transaction endpoints, AWS Secrets Manager, DynamoDB.
- Produces: `GET /v1/faucet/info`, `POST /v1/faucet/request`, `POST /v1/faucet/challenge`, plus the existing public RPC proxy routes.

- [ ] **Step 1: Import only the already-tested hardened Lambda files**

Copy the matching files from `codex/faucet-hardening-integration`, not the entire diverged branch.

- [ ] **Step 2: Run Lambda tests**

```bash
cd deployment/testnet/public-rpc/lambda
npm install --ignore-scripts --no-audit --no-fund
npm test
```

Expected: PASS.

- [ ] **Step 3: Run tracked-secret safety test**

```bash
bash ./scripts/check-tracked-secrets_test.sh
```

Expected: PASS.

- [ ] **Step 4: Commit the restored slice**

```bash
git add deployment/testnet/public-rpc/lambda
git commit -m "feat(faucet): restore hardened public rpc slice"
```

### Task 3: Make artifact promotion manual and same-run

**Files:**
- Create: `.github/workflows/testnet-public-rpc.yml`
- Test: `scripts/check-faucet-deploy-contract_test.sh`

**Interfaces:**
- Consumes: Lambda source and successful build artifact from the same workflow run.
- Produces: a manual-only AWS deploy path guarded by an explicit boolean input.

- [ ] **Step 1: Implement build/test/package on branch push**

The build job runs `npm test`, packages the Lambda, and uploads `sudharma-testnet-wallet-proxy`.

- [ ] **Step 2: Implement explicit manual deploy gate**

Use:

```yaml
workflow_dispatch:
  inputs:
    deploy:
      description: Deploy the tested artifact to the public testnet Lambda
      required: true
      default: false
      type: boolean
```

and gate AWS jobs with:

```yaml
if: github.event_name == 'workflow_dispatch' && inputs.deploy == true
```

The deploy job must use `actions/download-artifact@v4`; it must not rebuild the Lambda after the build job.

- [ ] **Step 3: Preserve staged activation and rollback**

Deploy code, set `FAUCET_ENABLED=false`, verify `/v1/status`, set `FAUCET_ENABLED=true`, call `/v1/faucet/info`, and restore `FAUCET_ENABLED=false` before failing if the info smoke test is not HTTP 200.

- [ ] **Step 4: Run regression contract test**

```bash
bash ./scripts/check-faucet-deploy-contract_test.sh
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/testnet-public-rpc.yml scripts/check-faucet-deploy-contract_test.sh
git commit -m "fix(faucet): make tested artifact promotion explicit"
```

### Task 4: Verification checkpoint

**Files:** no production changes.

**Interfaces:**
- Consumes: Stage 2 branch head.
- Produces: evidence required before any merge or deployment request.

- [ ] **Step 1: Run all Stage 2 faucet tests**

```bash
bash ./scripts/check-faucet-deploy-contract_test.sh
cd deployment/testnet/public-rpc/lambda && npm test
```

Expected: all PASS.

- [ ] **Step 2: Inspect GitHub Actions result**

Require the Stage 2 faucet recovery CI run to finish successfully.

- [ ] **Step 3: Stop before deployment**

Report the branch SHA, files changed, tests passed, and remaining AWS/OIDC prerequisites. Do not merge and do not run the manual deployment until explicit approval is given.
