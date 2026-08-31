# Canonical Wallet, Faucet, and Demand-Miner Integration Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce one exact, CI-verifiable wallet/faucet/demand-miner candidate while preserving the latest challenge reconciliation and keeping deployment and GPU consensus disabled.

**Architecture:** Continue from `feature/public-testnet-wallet-v2`, treating PR #23 as the combined review line. Reconcile only independently tested demand-miner hardening from `feature/demand-miner-v1`; preserve the canonical faucet/challenge commits and validate every resulting tree at the exact commit.

**Tech Stack:** Go, Kotlin/Gradle Android, Node.js built-in test runner, Bash, GitHub Actions, AWS Lambda deployment templates.

**Spec:** `docs/superpowers/specs/2026-08-28-canonical-wallet-faucet-integration-design.md`

## Global Constraints

- Mainnet remains disabled.
- Do not change genesis, zero premine, supply, subsidy, fee calculation, or active consensus.
- Do not merge, activate, or deploy GPU-PoW/Khushi consensus.
- Do not deploy to Seed-1 or Seed-2 in this plan.
- Do not add public mining controls or broaden AWS/IAM authority.
- Keep PRs #20, #21, #23, and #25 draft.
- Preserve wallet-local signing and secret-safe repository/runtime boundaries.
- The local runtime has no Go toolchain; record this and use exact-head GitHub Actions for Go verification.

---

### Task 1: Record Branch-Divergence Classification

**Files:**
- Create: `docs/audits/2026-08-28-wallet-demand-miner-divergence.md`

**Interfaces:**
- Consumes: `origin/feature/public-testnet-wallet-v2`, `origin/feature/demand-miner-v1`, and `origin/main` commit graphs.
- Produces: an evidence table mapping every unique demand-miner hardening pair to its target files and integration decision.

- [ ] **Step 1: Capture immutable branch heads and patch-unique history**

Run:

```bash
git rev-parse origin/main origin/feature/public-testnet-wallet-v2 origin/feature/demand-miner-v1
git log --left-right --cherry-pick --oneline origin/feature/public-testnet-wallet-v2...origin/feature/demand-miner-v1
```

Expected heads include `5c925b4`, `76ae28e`, and `bef507c`; the demand-miner branch is 41 commits ahead and 10 commits behind the canonical base.

- [ ] **Step 2: Classify unique hardening**

Write the audit with these explicit candidates:

| RED | GREEN | Required behavior |
|---|---|---|
| `e6faee5` | `8429f97` | stop the unique mining child only after positive pending-transaction and post-broadcast block evidence |
| `c47affe` | `e08a48a` | retain bounded diagnostics while recognizing late mining evidence |
| `3373310` | `ad997c5` | never overwrite `/usr/local/bin/sudharmad`; install the reviewed child under the miner-only libexec path |
| `9de2027` | `baeed07` | keep a stable runtime lock inode in a systemd-owned runtime directory |
| `a4328b0` | `9ee2b21` | reject `DESTDIR + --enable` before any staging mutation |
| `28a2dd6` | `cd0939f` | reject symlink and non-regular configuration targets before mutation |
| `a3f9a98` | `bef507c` | reject invalid seed peer ports during configuration validation |

Also classify `f76f07d` as deterministic-test hardening and `6d4ad86` as test-only asynchronous peer admission synchronization.

- [ ] **Step 3: Verify the audit contains no deployment authorization**

Run:

```bash
rg -n "deploy|enable|Seed-1|Seed-2|GPU|Mainnet" docs/audits/2026-08-28-wallet-demand-miner-divergence.md
git diff --check
```

Expected: deployment references are explicitly blocked; no whitespace errors.

- [ ] **Step 4: Commit**

```bash
git add docs/audits/2026-08-28-wallet-demand-miner-divergence.md
git commit -m "docs(testnet): classify demand miner divergence"
```

### Task 2: Integrate Runner Lifecycle Hardening

**Files:**
- Modify: `demandminer/runner.go`
- Modify: `demandminer/runner_test.go`

**Interfaces:**
- Consumes: canonical `NativeRunner` process lifecycle and bounded child-output contract.
- Produces: deterministic recognition of positive pending and post-broadcast evidence without unbounded output retention.

- [ ] **Step 1: Apply the lifecycle RED/GREEN history without committing automatically**

```bash
git cherry-pick -n e6faee5 8429f97 f76f07d c47affe e08a48a
```

Resolve conflicts by retaining canonical public-testnet code and the demand-miner branch's focused runner tests/implementation. Do not accept changes outside `demandminer/runner.go` and `demandminer/runner_test.go` from this step.

- [ ] **Step 2: Inspect the resulting contract**

```bash
git diff -- demandminer/runner.go demandminer/runner_test.go
rg -n "64|evidence|Transactions:|pending|cancel|deadline" demandminer/runner.go demandminer/runner_test.go
git diff --check
```

Expected: diagnostics remain capped; evidence recognition has a separate bounded window; child cancellation occurs only after both evidence classes.

- [ ] **Step 3: Run available static checks**

```bash
bash scripts/check-demand-miner-ci_test.sh
```

If the canonical branch uses a different assertion script name, run the tracked script that verifies the same CI sources and record the exact command in the commit message body.

- [ ] **Step 4: Commit**

```bash
git add demandminer/runner.go demandminer/runner_test.go
git commit -m "fix(testnet): harden bounded miner lifecycle"
```

### Task 3: Integrate Installer Isolation and Lock Safety

**Files:**
- Modify: `deployment/testnet/install-demand-miner.sh`
- Modify: `deployment/testnet/install-demand-miner_test.sh`
- Modify: `deployment/testnet/sudharma-demand-miner.service`
- Modify: `deployment/testnet/demand-miner.example.json`
- Modify: `deployment/testnet/README.md`

**Interfaces:**
- Consumes: built `sudharma-demand-miner` and reviewed `sudharmad` child binary.
- Produces: disabled-by-default, idempotent packaging that preserves the shared node installation and uses `/usr/local/libexec/sudharma-demand-miner/sudharmad` plus `/run/sudharma-demand-miner/lock`.

- [ ] **Step 1: Apply shared-binary and lock RED/GREEN pairs without committing**

```bash
git cherry-pick -n 3373310 ad997c5 09b4381 d58c858 9de2027 5a4afdd baeed07
```

- [ ] **Step 2: Run installer safety tests**

```bash
bash deployment/testnet/install-demand-miner_test.sh
```

Expected: PASS; a fixture at `/usr/local/bin/sudharmad` under the staged root remains byte-identical, the miner child uses the libexec path, and the runtime lock path is stable.

- [ ] **Step 3: Check service hardening and rollback scope**

```bash
rg -n "RuntimeDirectory|NoNewPrivileges|ProtectSystem|PrivateTmp|ReadWritePaths|libexec|/run/sudharma-demand-miner/lock" deployment/testnet
! rg -n "rm .*var/lib/sudharma|overwrite.*/usr/local/bin/sudharmad" deployment/testnet/install-demand-miner.sh deployment/testnet/README.md
git diff --check
```

Expected: hardened directives exist; rollback does not remove node state or overwrite the shared executable.

- [ ] **Step 4: Commit**

```bash
git add deployment/testnet
git commit -m "ops(testnet): isolate demand miner installation"
```

### Task 4: Integrate Side-Effect-Free Preflight Validation

**Files:**
- Modify: `deployment/testnet/install-demand-miner.sh`
- Modify: `deployment/testnet/install-demand-miner_test.sh`
- Modify: `demandminer/config.go`
- Modify: `demandminer/config_test.go`

**Interfaces:**
- Consumes: installer target paths and demand-miner JSON configuration.
- Produces: mutation-free rejected activation, safe regular-file configuration handling, and strict seed `host:port` validation.

- [ ] **Step 1: Apply preflight RED/GREEN pairs without committing**

```bash
git cherry-pick -n a4328b0 9ee2b21 28a2dd6 cd0939f a3f9a98 bef507c
```

- [ ] **Step 2: Run installer tests**

```bash
bash deployment/testnet/install-demand-miner_test.sh
```

Expected: PASS for refused staged enablement with an unchanged root and PASS for rejecting symlink/non-regular config targets before mutation.

- [ ] **Step 3: Verify strict port cases exist**

```bash
rg -n "port|65535|65536|zero|negative|seed" demandminer/config_test.go
git diff --check
```

Expected: tests cover missing, non-numeric, zero, negative, and greater-than-65535 ports.

- [ ] **Step 4: Commit**

```bash
git add demandminer/config.go demandminer/config_test.go deployment/testnet/install-demand-miner.sh deployment/testnet/install-demand-miner_test.sh
git commit -m "fix(testnet): reject unsafe miner configuration"
```

### Task 5: Integrate Deterministic P2P Test Synchronization

**Files:**
- Modify: `p2p/block_duplicate_gossip_test.go`

**Interfaces:**
- Consumes: existing `waitForPeerCount` test helper.
- Produces: triangle gossip test that waits for asynchronous inbound handshakes without changing production P2P behavior.

- [ ] **Step 1: Apply the test-only commit without committing**

```bash
git cherry-pick -n 6d4ad86
```

- [ ] **Step 2: Prove only the test changed**

```bash
git diff --name-only
git diff -- p2p/block_duplicate_gossip_test.go
git diff --check
```

Expected: this task changes only `p2p/block_duplicate_gossip_test.go` and calls the existing wait helper.

- [ ] **Step 3: Commit**

```bash
git add p2p/block_duplicate_gossip_test.go
git commit -m "test(p2p): await asynchronous triangle peers"
```

### Task 6: Verify Faucet and 25-to-50 Challenge Contracts

**Files:**
- Modify only if a test fails: `deployment/testnet/public-rpc/lambda/faucet.mjs`
- Test: `deployment/testnet/public-rpc/lambda/faucet.test.mjs`
- Test: `deployment/testnet/public-rpc/lambda/faucet-runtime.test.mjs`
- Test: `deployment/testnet/public-rpc/lambda/index.test.mjs`
- Test: `deployment/testnet/public-rpc/lambda/router.test.mjs`
- Test: `deployment/testnet/public-rpc/lambda/upstream.test.mjs`

**Interfaces:**
- Consumes: confirmed transaction lookup, faucet signer boundary, and DynamoDB reservation runtime.
- Produces: one-time 100 Test SUDH grant and replay-safe confirmed 25-to-50 challenge behavior.

- [ ] **Step 1: Run Lambda tests at the integrated candidate**

```bash
cd deployment/testnet/public-rpc/lambda
npm test
```

Expected: all tests pass, including automatic initial-grant reconciliation, exact sender/recipient/25 amount, 50 reward, cooldown, five-round maximum, duplicate transaction rejection, uncertain submission, failover, and sanitized errors.

- [ ] **Step 2: Verify constants and Android metadata consumption agree**

```bash
rg -n "CHALLENGE_SEND_SUDH|CHALLENGE_REWARD_SUDH|challenge_send_sudh|challenge_reward_sudh" deployment/testnet/public-rpc/lambda mobile/android/app/src
```

Expected: server constants are 25 and 50; Android consumes live metadata and validates the challenge address.

- [ ] **Step 3: Verify the challenge reconciliation commits remain ancestors**

```bash
git merge-base --is-ancestor 0b19679 HEAD
git merge-base --is-ancestor fc8ad51 HEAD
```

Expected: both commands exit 0.

- [ ] **Step 4: Commit only if a reproducible failure required a minimal fix**

```bash
git add deployment/testnet/public-rpc/lambda
git commit -m "fix(faucet): preserve confirmed challenge contract"
```

If no fix was needed, record the passing command in the divergence audit and do not create an empty commit.

### Task 7: Run Android and Repository-Available Verification

**Files:**
- Modify only if a test fails: files directly responsible for that failure.
- Update: `docs/audits/2026-08-28-wallet-demand-miner-divergence.md`

**Interfaces:**
- Consumes: integrated candidate tree.
- Produces: local evidence plus a documented list of checks deferred to exact-head GitHub Actions.

- [ ] **Step 1: Run Android verification when the wrapper/toolchain is available**

```bash
cd mobile/android
./gradlew testDebugUnitTest lintDebug assembleDebug
```

Expected: unit tests, lint, and debug APK build pass. If the tracked project has no wrapper, record that exact limitation and rely on the Android Wallet GitHub workflow.

- [ ] **Step 2: Run shell and Node verification**

```bash
cd ../..
bash deployment/testnet/install-demand-miner_test.sh
bash scripts/check-demand-miner-ci_test.sh
(cd deployment/testnet/public-rpc/lambda && npm test)
```

Expected: all locally available checks pass.

- [ ] **Step 3: Record unavailable Go commands without claiming success**

Record these required GitHub CI commands in the audit:

```bash
gofmt -l .
go test ./... -count=1
go vet ./...
go build -trimpath ./cmd/sudharma-demand-miner ./cmd/sudharmad
go test -race ./demandminer ./cmd/sudharma-demand-miner -count=1
go test -race ./... -count=1
```

- [ ] **Step 4: Commit evidence**

```bash
git add docs/audits/2026-08-28-wallet-demand-miner-divergence.md
git commit -m "docs(testnet): record canonical verification"
```

### Task 8: Publish Candidate and Require Exact-Head CI

**Files:**
- Modify: PR #23 description only after the candidate is pushed.

**Interfaces:**
- Consumes: clean canonical integration branch and local verification evidence.
- Produces: one remote candidate commit with exact GitHub Actions evidence; no deployment.

- [ ] **Step 1: Verify clean repository and safety boundaries**

```bash
git status --short
git diff origin/feature/public-testnet-wallet-v2...HEAD -- pow params compatibility/cuda compatibility/opencl
rg -n "mainnet.*enabled|AdministratorAccess|AKIA|BEGIN .*PRIVATE KEY" --glob '!*.sum' .
```

Expected: clean status; no GPU consensus files changed; no Mainnet activation, broad IAM policy, AWS access key, or private key is introduced.

- [ ] **Step 2: Push the candidate branch**

```bash
git push -u origin codex/canonical-wallet-integration
```

- [ ] **Step 3: Open or update a draft PR targeting `feature/public-testnet-wallet-v2`**

The PR body must list the exact head, integrated hardening pairs, local test evidence, unavailable local Go toolchain, and explicit no-deployment/no-GPU boundary.

- [ ] **Step 4: Verify exact-head workflows**

Require successful completion of general CI and Android Wallet workflows at the exact candidate SHA. A passing earlier SHA does not satisfy this gate. Do not rerun or deploy automatically in response to a genuine deterministic failure; diagnose and fix it through a new RED/GREEN commit.

- [ ] **Step 5: Update PR #23 only after exact-head success**

Document the candidate branch/PR and CI run URLs in PR #23 while keeping it draft and explicitly undeployed.
