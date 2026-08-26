# Public RPC Wallet Autosync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a least-privilege AWS-hosted public Testnet wallet API with two-seed private failover, compile its stable HTTPS endpoint into the Android Testnet wallet, integrate the official Sudharma logo, and close the remaining Android release blockers without enabling Mainnet or exposing raw RPC.

**Architecture:** API Gateway HTTP API invokes one VPC-attached Go Lambda. The Lambda exposes only an explicit allowlist of wallet endpoints and forwards to a dedicated private Nginx listener on each seed at TCP `29100`; Nginx proxies only approved wallet paths to each node's localhost RPC at `127.0.0.1:28545`. Read requests fail over between synchronized seeds; transaction submission may retry only the exact same signed payload and deterministic transaction ID. Android receives the stable `execute-api` HTTPS base URL at build time, migrates blank legacy configuration automatically, and shows connection state rather than a user-facing RPC form.

**Tech Stack:** Go, AWS Lambda (`provided.al2023`), API Gateway HTTP API, CloudFormation/SAM-style template, GitHub Actions OIDC, Nginx, Kotlin, Jetpack Compose, OkHttp/Moshi, AndroidX lifecycle/saved-state, Android instrumentation tests.

**Spec:** `docs/superpowers/specs/2026-08-26-public-rpc-wallet-autosync-design.md` (approved locally in commit `3cbff9a`; verify/cherry-pick that exact local commit before implementation rather than recreating it blindly).

## Global Constraints

- Never read, modify, delete, replace, reset, or overwrite `C:\\sudh`.
- Never request, expose, or persist wallet secrets, recovery phrases, private keys, passwords, AWS access keys, secret keys, MFA codes, signing keystores, or treasury material.
- AWS deployments use GitHub OIDC role `arn:aws:iam::981626123397:role/Sudharma-GitHub-Actions-Testnet`; no permanent AWS credentials and no `AdministratorAccess`.
- Lambda runtime role is `arn:aws:iam::981626123397:role/Sudharma-Wallet-Proxy-Lambda-Execution`.
- VPC is `vpc-0cd862d72cf8165fa`.
- Seed-1: instance `i-06e7ddb174e4941de`, private IP `172.31.10.171`, subnet `subnet-0c04f19843bf6a401`, SG `sg-09f1c1cf738869177`, AZ `ap-south-1b`.
- Seed-2: instance `i-07422df89342dd5f9`, private IP `172.31.32.195`, subnet `subnet-0f213d036a08fd543`, SG `sg-07f0487f1de24caed`, AZ `ap-south-1a`.
- Dedicated Lambda SG is `sg-057c9893359ab2300`.
- Existing TCP `29090` is reserved for monitoring/metrics; do not use it as the wallet proxy path.
- Raw RPC remains bound to `127.0.0.1:28545` and must never be opened publicly.
- Use dedicated private wallet-proxy TCP port `29100`; seed SG inbound source must be only `sg-057c9893359ab2300`.
- Public API allowlist is exactly: `GET /health`, `GET /ready`, `GET /v1/status`, `GET /v1/accounts/{address}`, `POST /v1/transactions`, `GET /v1/transactions/{transactionID}`.
- Do not expose `/metrics`, raw RPC, administrative endpoints, block enumeration, or mempool enumeration.
- Mainnet remains unavailable.
- PR #20 stays open and draft until every release gate and independent review passes.

---

### Task 1: Correct and lock the private seed ingress path

**Files:**
- Create: `deployment/testnet/public-rpc/nginx-wallet-proxy.conf`
- Create: `deployment/testnet/public-rpc/nginx-wallet-proxy_test.go`
- Modify: `deployment/testnet/README.md`

**Interfaces:**
- Consumes: node localhost RPC at `127.0.0.1:28545`.
- Produces: private HTTP listener on `0.0.0.0:29100` that accepts only the six approved routes and returns `404` for every other path.

- [ ] **Step 1: Write the failing configuration test**

Create a Go test that loads `nginx-wallet-proxy.conf` and asserts it contains `listen 29100`, `proxy_pass http://127.0.0.1:28545`, exact route locations for the six approved API shapes, `client_max_body_size 1m`, bounded proxy timeouts, `Cache-Control "no-store"`, and no `/metrics`, `/blocks`, or `/mempool` location.

- [ ] **Step 2: Run the test and verify failure**

Run: `go test ./deployment/testnet/public-rpc -run TestNginxWalletProxyPolicy -count=1`

Expected: FAIL because `nginx-wallet-proxy.conf` does not yet exist.

- [ ] **Step 3: Add the minimal Nginx private proxy config**

Implement a server listening on `29100` with `server_name _;`, `client_max_body_size 1m`, `proxy_connect_timeout 2s`, `proxy_read_timeout 8s`, `proxy_send_timeout 8s`, `add_header Cache-Control "no-store" always;`, explicit exact/prefix locations for the approved wallet routes, and a final `location / { return 404; }`. Do not configure TLS on this private VPC hop; API Gateway provides public TLS.

- [ ] **Step 4: Run the test and Nginx syntax check**

Run: `go test ./deployment/testnet/public-rpc -run TestNginxWalletProxyPolicy -count=1`

On each seed after owner-approved deployment, run: `sudo nginx -t`.

Expected: PASS and `syntax is ok`.

- [ ] **Step 5: Correct AWS security groups**

Remove the temporary Lambda-SG-to-seed `29090` permissions that were added during discovery. Add Lambda SG egress TCP `29100` only to `172.31.10.171/32` and `172.31.32.195/32`. On Seed-1 and Seed-2 SGs, add inbound TCP `29100` with source SG `sg-057c9893359ab2300`, then remove the temporary Lambda-source inbound `29090` rules. Preserve all pre-existing monitoring `29090`, SSH, and P2P rules.

- [ ] **Step 6: Verify private reachability and public non-reachability**

From an authorized VPC test context, verify `http://172.31.10.171:29100/ready` and `http://172.31.32.195:29100/ready` return ready. Verify the seeds' public IPs do not expose `29100` or `28545`.

- [ ] **Step 7: Commit**

```bash
git add deployment/testnet/public-rpc deployment/testnet/README.md
git commit -m "feat(testnet): add private wallet RPC proxy"
```

### Task 2: Build the Lambda route allowlist and bounded proxy core

**Files:**
- Create: `cmd/sudharma-wallet-proxy/main.go`
- Create: `publicrpc/handler.go`
- Create: `publicrpc/handler_test.go`
- Create: `publicrpc/routes.go`
- Create: `publicrpc/routes_test.go`

**Interfaces:**
- Consumes: API Gateway HTTP API v2 events and seed base URLs `http://172.31.10.171:29100`, `http://172.31.32.195:29100`.
- Produces: API Gateway v2 responses with `Cache-Control: no-store`; maximum request body 1 MiB; maximum upstream response 4 MiB.

- [ ] **Step 1: Write failing allowlist tests**

Cover all six approved method/path combinations and rejection of `/metrics`, `/v1/blocks`, `/v1/mempool`, unknown methods, malformed account addresses, and malformed 64-hex transaction IDs.

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./publicrpc -run 'TestRoute|TestValidation' -count=1`

Expected: FAIL because router symbols are undefined.

- [ ] **Step 3: Implement route parsing and validation**

Define `type RouteKind int`, `func MatchRoute(method, path string) (RouteKind, map[string]string, error)`, `func ValidAddress(string) bool`, and `func ValidTransactionID(string) bool`. Reject encoded path traversal and any path outside the allowlist.

- [ ] **Step 4: Write failing handler boundary tests**

Use `httptest.Server` seeds to assert request-size rejection (`413`), upstream timeout mapping (`504`), oversized response rejection (`502`), no-store headers on success/error, and safe logs that omit request bodies and authorization-like headers.

- [ ] **Step 5: Implement minimal handler**

Create an `http.Client` with bounded connect/overall timeouts, copy only safe headers, never log transaction bodies, and return generic upstream errors without internal IPs.

- [ ] **Step 6: Run package tests**

Run: `go test ./publicrpc ./cmd/sudharma-wallet-proxy -count=1`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add publicrpc cmd/sudharma-wallet-proxy
git commit -m "feat: add bounded public wallet proxy"
```

### Task 3: Add synchronized two-seed read failover and exact transaction retry

**Files:**
- Create: `publicrpc/failover.go`
- Create: `publicrpc/failover_test.go`
- Modify: `publicrpc/handler.go`

**Interfaces:**
- Consumes: two seed clients and the exact incoming signed transaction bytes.
- Produces: deterministic seed selection/failover; transaction retry reuses byte-for-byte identical payload and expected transaction ID.

- [ ] **Step 1: Write failing read-failover tests**

Verify primary success does not call secondary; transport/5xx failure calls secondary; 4xx validation responses do not fail over; `/v1/status` results with incompatible network/tip identity mark service degraded instead of pretending synchronization.

- [ ] **Step 2: Write failing transaction retry tests**

Send a fixed signed JSON body containing deterministic transaction ID `aaaaaaaa...` (64 hex chars). Make seed-1 accept then disconnect before response; assert seed-2 receives the exact same bytes. Assert no code path mutates nonce, fee, ID, signature, or creates a replacement transaction.

- [ ] **Step 3: Implement `SeedPool`**

Define `type SeedPool struct { Seeds []*SeedClient }`, `func (p *SeedPool) DoRead(...)`, and `func (p *SeedPool) SubmitExact(ctx context.Context, body []byte, expectedID string)`. Retry only transport errors/5xx where outcome is uncertain and only with identical bytes.

- [ ] **Step 4: Add uncertain-outcome semantics**

If all submission attempts fail without authoritative acceptance/rejection, return a retryable/uncertain gateway error; never return success. Android will reconcile using `GET /v1/transactions/{transactionID}`.

- [ ] **Step 5: Run tests**

Run: `go test ./publicrpc -run 'TestFailover|TestSubmitExact|TestUncertain' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add publicrpc
git commit -m "feat: add two-seed safe failover"
```

### Task 4: Define least-privilege AWS infrastructure and OIDC deployment policy

**Files:**
- Create: `deployment/testnet/public-rpc/template.yaml`
- Create: `deployment/testnet/public-rpc/iam-github-actions-policy.json`
- Create: `deployment/testnet/public-rpc/iam-policy_test.go`
- Create: `.github/workflows/deploy-testnet-wallet-proxy.yml`

**Interfaces:**
- Consumes: GitHub OIDC role `Sudharma-GitHub-Actions-Testnet`, Lambda runtime role ARN, VPC/subnets/SG IDs.
- Produces: one Lambda, one HTTP API, one stage, log group/retention, throttling, alarms, and deployment workflow restricted to branch `feature/android-wallet-v0.1`.

- [ ] **Step 1: Write failing IAM-policy tests**

Assert the policy contains no `*` action, no `AdministratorAccess`, no IAM user/access-key actions, and scopes writes to named Sudharma wallet-proxy Lambda/API/CloudWatch resources. Allow only the minimum deploy-time actions required by the selected deployment mechanism plus `iam:PassRole` restricted to `Sudharma-Wallet-Proxy-Lambda-Execution` and `lambda.amazonaws.com`.

- [ ] **Step 2: Write failing infrastructure-template tests**

Assert Lambda uses both seed subnets, SG `sg-057c9893359ab2300`, runtime role ARN, memory/timeout caps, reserved concurrency, structured log retention, HTTP API throttling, and only the six routes. Assert no `/metrics` route and no Mainnet value.

- [ ] **Step 3: Implement template and policy**

Set environment variables only for non-secret configuration: seed private URLs, network=`testnet`, response/request limits. Add CloudWatch alarms for Lambda errors, throttles, duration, API 5xx, and a custom degraded-seed metric.

- [ ] **Step 4: Implement OIDC workflow**

Use `permissions: id-token: write, contents: read`, `aws-actions/configure-aws-credentials` with role ARN `arn:aws:iam::981626123397:role/Sudharma-GitHub-Actions-Testnet`, region `ap-south-1`, and no static secrets. Build/test before deploy.

- [ ] **Step 5: Run policy/template tests and secret guard**

Run: `go test ./deployment/testnet/public-rpc -count=1` and the repository's tracked-secret guard.

Expected: PASS.

- [ ] **Step 6: Owner review and attach only the custom deployment policy**

Review JSON against exact created resource names before attaching it to `Sudharma-GitHub-Actions-Testnet`. Do not attach broad AWS managed policies.

- [ ] **Step 7: Commit**

```bash
git add deployment/testnet/public-rpc .github/workflows/deploy-testnet-wallet-proxy.yml
git commit -m "ci: deploy testnet wallet proxy with OIDC"
```

### Task 5: Deploy and smoke-test the public gateway

**Files:**
- Create: `scripts/smoke-public-wallet-api.sh`
- Create: `deployment/testnet/public-rpc/OPERATIONS.md`

**Interfaces:**
- Consumes: deployed `https://<api-id>.execute-api.ap-south-1.amazonaws.com` endpoint.
- Produces: verified stable Testnet HTTPS base URL and operational runbook.

- [ ] **Step 1: Write smoke script assertions**

Check `/health`, `/ready`, `/v1/status`, account validation, transaction-ID validation, `Cache-Control: no-store`, request-size rejection, and explicit `404/405` for blocked endpoints.

- [ ] **Step 2: Deploy from OIDC workflow**

Verify CloudFormation/Lambda/API resources are tagged `Project=Sudharma-Testnet` and no unexpected resources are created.

- [ ] **Step 3: Test seed failover**

Temporarily make one private seed target unavailable using an owner-approved reversible method, confirm reads remain available/degraded, restore it, and confirm synchronized state. Do not stop both seeds simultaneously.

- [ ] **Step 4: Test transaction uncertainty safely**

Use a non-secret Testnet fixture transaction only; verify the gateway never generates a replacement and reports uncertain outcome rather than false success when both upstream attempts fail.

- [ ] **Step 5: Commit runbook/smoke script**

```bash
git add scripts/smoke-public-wallet-api.sh deployment/testnet/public-rpc/OPERATIONS.md
git commit -m "test: add public wallet API smoke checks"
```

### Task 6: Compile the Testnet endpoint and migrate blank Android preferences

**Files:**
- Modify: `mobile/android/app/build.gradle.kts`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletPreferences.kt`
- Create: `mobile/android/app/src/test/java/network/sudharma/wallet/WalletPreferencesTest.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/SudharmaWalletRepository.kt`

**Interfaces:**
- Consumes: build-time `TESTNET_API_BASE_URL` HTTPS value.
- Produces: `WalletPreferences.effectiveRpcUrl()` using compiled endpoint when legacy preference is blank; debug-only override remains possible.

- [ ] **Step 1: Write failing migration tests**

Verify a fresh install and an existing blank `testnet_rpc_url` use the compiled endpoint; an existing explicit HTTPS value is preserved only in debug builds; release builds ignore/clear manual overrides; invalid/non-HTTPS compiled endpoint fails build/config validation.

- [ ] **Step 2: Run test and verify failure**

Run: `cd mobile/android && ./gradlew testDebugUnitTest --tests '*WalletPreferencesTest*'`

- [ ] **Step 3: Implement build config and migration**

Expose `BuildConfig.TESTNET_API_BASE_URL`. Replace direct `rpcUrl` UI dependency with `effectiveRpcUrl()`. Keep Mainnet absent.

- [ ] **Step 4: Remove ordinary-user RPC form**

In production UI flow, show connection status/height/last sync only. Keep developer override behind `BuildConfig.DEBUG`.

- [ ] **Step 5: Run Android tests**

Run: `cd mobile/android && ./gradlew testDebugUnitTest lintDebug`

- [ ] **Step 6: Commit**

```bash
git add mobile/android
git commit -m "feat(android): auto-configure testnet API"
```

### Task 7: Add autosync state and RPC response integrity checks

**Files:**
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/PortfolioState.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/SudharmaWalletRepository.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/chain/sudharma/SudharmaRpcClient.kt`
- Modify: `mobile/android/app/src/test/java/network/sudharma/wallet/PortfolioStateTest.kt`
- Modify: `mobile/android/app/src/test/java/network/sudharma/wallet/chain/sudharma/SudharmaRpcClientTest.kt`

**Interfaces:**
- Produces connection states `CONNECTING`, `SYNCHRONIZED`, `DEGRADED`, `OFFLINE` plus height and last-sync timestamp.

- [ ] **Step 1: Write failing autosync state tests**

Cover initial connecting, successful synchronized status, one-seed degraded signal, bounded exponential backoff, offline after repeated transport failures, and recovery without app restart.

- [ ] **Step 2: Write failing response-integrity tests**

For every response shape carrying an ID or request correlation field, reject missing/mismatched IDs rather than accepting a different response. Preserve the existing transaction-ID equality check in `SudharmaChainAdapter.submit`.

- [ ] **Step 3: Implement repository autosync loop**

Use lifecycle-aware coroutine scope, bounded exponential backoff with jitter and a maximum delay, no busy-looping, and truthful status transitions.

- [ ] **Step 4: Route production portfolio through `PortfolioState`**

Make home/activity rendering consume `PortfolioState` instead of parallel ad-hoc state.

- [ ] **Step 5: Run tests**

Run: `cd mobile/android && ./gradlew testDebugUnitTest`

- [ ] **Step 6: Commit**

```bash
git add mobile/android
git commit -m "feat(android): add truthful testnet autosync state"
```

### Task 8: Make prepared transfers immutable and recreation-safe

**Files:**
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/chain/ChainModels.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/chain/sudharma/SudharmaChainAdapter.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt`
- Create: `mobile/android/app/src/main/java/network/sudharma/wallet/PreparedTransferState.kt`
- Create: `mobile/android/app/src/test/java/network/sudharma/wallet/PreparedTransferStateTest.kt`
- Modify: `mobile/android/app/src/test/java/network/sudharma/wallet/chain/sudharma/SudharmaTransferFlowTest.kt`

**Interfaces:**
- Produces immutable `PreparedTransfer(from,to,amountAtomic,feeAtomic,nonce)` before authorization; signing consumes that exact object; persisted send state records transaction ID and submission phase.

- [ ] **Step 1: Write failing prepared-transfer tests**

Assert fee and next nonce are fetched before confirmation/authorization, the confirmation displays those exact immutable values, and signing cannot recalculate fee or nonce afterward.

- [ ] **Step 2: Write failing recreation tests**

Simulate recreation in phases `EDITING`, `PREPARED`, `SIGNED_NOT_SUBMITTED`, `SUBMITTING`, `SUBMITTED_PENDING`. Assert recreation never auto-signs a new transaction and never submits a different transaction ID.

- [ ] **Step 3: Implement immutable prepared state**

Persist only non-secret transfer metadata and signed transaction payload only if existing encrypted app storage boundary permits it; otherwise persist transaction ID/submission phase and require explicit user reconciliation. Never put private keys in saved state.

- [ ] **Step 4: Implement ambiguous-submit reconciliation**

After uncertain network outcome, query status by the existing transaction ID before offering retry. A retry sends the exact same signed bytes.

- [ ] **Step 5: Run tests**

Run: `cd mobile/android && ./gradlew testDebugUnitTest --tests '*PreparedTransferStateTest*' --tests '*SudharmaTransferFlowTest*'`

- [ ] **Step 6: Commit**

```bash
git add mobile/android
git commit -m "fix(android): make send flow recreation safe"
```

### Task 9: Move payment URI parsing behind the chain adapter and use production backup boundary

**Files:**
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/chain/ChainModels.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/chain/sudharma/SudharmaChainAdapter.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/chain/sudharma/SudharmaPaymentUri.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/backup/CloudBackupBoundary.kt`
- Modify: `mobile/android/app/src/test/java/network/sudharma/wallet/chain/sudharma/SudharmaPaymentUriTest.kt`
- Modify: `mobile/android/app/src/test/java/network/sudharma/wallet/backup/CloudBackupBoundaryTest.kt`

**Interfaces:**
- Produces `ChainAdapter.parsePaymentRequest(raw)` returning address plus optional amount; production optional backup flow must pass only encrypted ciphertext through `CloudBackupBoundary`.

- [ ] **Step 1: Write failing QR amount tests**

Assert scanning a valid Sudharma URI with amount pre-fills both address and amount; invalid/overflow amount is rejected; plain address remains supported.

- [ ] **Step 2: Implement chain-level payment request parsing**

Move parsing out of Compose/UI code. UI consumes only chain-agnostic parsed result.

- [ ] **Step 3: Write failing production backup-boundary test**

Exercise the production backup action and assert plaintext recovery material cannot cross `CloudBackupBoundary`; only `EncryptedBackup` ciphertext is accepted.

- [ ] **Step 4: Wire production flow through `CloudBackupBoundary`**

Keep backup optional and non-blocking.

- [ ] **Step 5: Run tests and commit**

Run: `cd mobile/android && ./gradlew testDebugUnitTest`

```bash
git add mobile/android
git commit -m "fix(android): enforce chain and backup boundaries"
```

### Task 10: Relock on background, preserve onboarding state, and add instrumentation security tests

**Files:**
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/MainActivity.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletFlow.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt`
- Modify: `mobile/android/app/build.gradle.kts`
- Create: `mobile/android/app/src/androidTest/java/network/sudharma/wallet/LifecycleRelockTest.kt`
- Create: `mobile/android/app/src/androidTest/java/network/sudharma/wallet/FlagSecureTest.kt`
- Create: `mobile/android/app/src/androidTest/java/network/sudharma/wallet/OnboardingRecreationTest.kt`

**Interfaces:**
- Produces immediate lock when app backgrounds after wallet unlock; onboarding survives configuration/process recreation without storing secrets in plain saved state.

- [ ] **Step 1: Write instrumentation tests first**

Use ActivityScenario to move activity to background/foreground and assert wallet returns to unlock. Assert `WindowManager.LayoutParams.FLAG_SECURE` is set on wallet screens. Recreate during onboarding and assert flow resumes safely.

- [ ] **Step 2: Run tests and observe failure**

Run: `cd mobile/android && ./gradlew connectedDebugAndroidTest`

- [ ] **Step 3: Implement lifecycle relock**

Use lifecycle callbacks/process lifecycle to clear unlocked session state on background. Do not delete encrypted wallet data.

- [ ] **Step 4: Implement recreation-safe onboarding**

Move non-secret flow state to `SavedStateHandle`/saveable state; keep recovery phrase/private material only inside existing secure boundaries and require safe restart if secret-only transient state cannot be restored.

- [ ] **Step 5: Run instrumentation and unit tests**

Run: `cd mobile/android && ./gradlew testDebugUnitTest connectedDebugAndroidTest lintDebug`

- [ ] **Step 6: Commit**

```bash
git add mobile/android
git commit -m "fix(android): relock and preserve safe lifecycle state"
```

### Task 11: Integrate official Sudharma branding with asset tests

**Files:**
- Verify authoritative source: Library asset `sudharma-logo(1).png` versus repo `assets/sudharma-logo.png`; do not replace until byte/visual comparison is complete.
- Modify/Create Android drawable/mipmap assets under `mobile/android/app/src/main/res/`.
- Modify: `mobile/android/app/src/main/AndroidManifest.xml`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt`
- Create: `mobile/android/app/src/test/java/network/sudharma/wallet/BrandAssetTest.kt`

**Interfaces:**
- Produces full circular logo on splash/welcome/About, derived center emblem for launcher/adaptive icon, compact home header logo, persistent TESTNET labeling.

- [ ] **Step 1: Verify authoritative source asset**

Compare dimensions/hash/visuals of the user-supplied 1254x1254 RGBA original with `assets/sudharma-logo.png`. If the Library asset is unavailable, ask the user to attach it again; do not guess.

- [ ] **Step 2: Write failing asset/config tests**

Assert launcher foreground/background resources exist, manifest references the intended icon, TESTNET text remains present, and full-logo resources are not accidentally replaced by the cropped emblem.

- [ ] **Step 3: Generate deterministic Android resources**

Keep original full logo untouched as source; derive center-emblem launcher asset with documented crop procedure. Do not alter brand colors/text beyond necessary small-icon crop.

- [ ] **Step 4: Integrate UI placements**

Add full logo to splash, welcome, Settings/About; compact logo to home/header. Preserve accessibility descriptions and clear TESTNET branding.

- [ ] **Step 5: Build and visually inspect**

Run debug build, capture emulator/device screenshots for launcher, splash, welcome, home, settings, and compare against approved source.

- [ ] **Step 6: Commit**

```bash
git add assets mobile/android
git commit -m "feat(android): integrate official Sudharma branding"
```

### Task 12: Full verification, device test, and PR release gate

**Files:**
- Modify: `.github/workflows/android-wallet.yml`
- Modify: PR #20 description/checklist only after verification; do not merge.

**Interfaces:**
- Produces a new intermediate Testnet APK plus hashes and an evidence checklist; not a final release until independent review passes.

- [ ] **Step 1: Run Go verification**

Run: `go test ./... -count=1`.

- [ ] **Step 2: Run Android verification**

Run: `cd mobile/android && ./gradlew testDebugUnitTest connectedDebugAndroidTest lintDebug assembleDebug`.

- [ ] **Step 3: Run secret guard**

Verify no tracked APK/AAB, keystore, `google-services.json`, `local.properties`, mnemonic/private key, AWS static credential, or secret-like material.

- [ ] **Step 4: Run deployed API smoke tests**

Execute `scripts/smoke-public-wallet-api.sh` against the stable `execute-api` URL and verify alarms/logging have no secret/body leakage.

- [ ] **Step 5: Hash APK**

Compute SHA-256 of the newly built APK and record it as an intermediate Testnet artifact.

- [ ] **Step 6: Install on OnePlus 11R and execute manual matrix**

Verify fresh install auto-connects without RPC entry; synchronized/degraded/offline states; background relock; rotation/recreation during onboarding/send; QR amount; prepared fee+nonce confirmation; exact retry semantics; screenshot blocking; backup boundary; logo placements; TESTNET branding.

- [ ] **Step 7: Independent review gate**

Have an independent reviewer verify all eight original blockers, infrastructure allowlist, least-privilege IAM, failover behavior, and no Mainnet path. Keep PR #20 draft/open if any item is unresolved.

- [ ] **Step 8: Update PR #20 evidence only**

Update the draft PR description with verified commit SHA, workflow results, API smoke evidence, APK hash, and explicit remaining limitations. Do not merge.
