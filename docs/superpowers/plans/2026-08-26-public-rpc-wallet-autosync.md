# Sudharma Public RPC Wallet Autosync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a Testnet Android wallet build that automatically connects through a stable AWS HTTPS endpoint, uses the official Sudharma logo, and resolves the known Android release blockers without exposing raw node RPC or wallet secrets.

**Architecture:** The public path is API Gateway HTTP API -> VPC Lambda (Node.js 24.x) -> two seed-private Nginx listeners on 29100 -> localhost node RPC on 28545. Android compiles the stable HTTPS endpoint, migrates blank configurations automatically, keeps all signing on-device, and surfaces truthful connection state. Backend and Android changes are developed test-first on isolated branches, with PR #20 remaining draft/open until release gates pass.

**Tech Stack:** Go tests for deployment policy, Node.js 24.x Lambda, AWS API Gateway HTTP API, AWS Lambda/VPC/security groups/CloudWatch, Android/Kotlin/Jetpack Compose, Gradle, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-08-26-public-rpc-wallet-autosync-design.md`

## Global Constraints

- Never read, modify, delete, replace, reset, or overwrite `C:\sudh`.
- Never request or expose recovery phrases, private keys, passwords, AWS access keys, secret keys, root credentials, MFA codes, signing keystores, or wallet secrets.
- Use AWS GitHub OIDC and short-lived roles only; never add permanent AWS credentials or `AdministratorAccess`.
- Mainnet remains disabled.
- Raw node RPC stays bound to `127.0.0.1:28545`.
- Public API permits only the six approved wallet routes from the spec.
- A transaction retry may resend only the exact same signed transaction body; never construct or sign a replacement transaction automatically.
- PR #20 remains draft/open and unmerged until all release blockers and independent review gates are complete.
- Preserve unrelated user changes and do not rewrite unrelated branch history.
- AWS account: `981626123397`.
- GitHub OIDC role: `arn:aws:iam::981626123397:role/Sudharma-GitHub-Actions-Testnet`.
- Lambda execution role: `arn:aws:iam::981626123397:role/Sudharma-Wallet-Proxy-Lambda-Execution`.
- VPC: `vpc-0cd862d72cf8165fa`.
- Seed-1: instance `i-06e7ddb174e4941de`, private IP `172.31.10.171`, subnet `subnet-0c04f19843bf6a401`, SG `sg-09f1c1cf738869177`, AZ `ap-south-1b`.
- Seed-2: instance `i-07422df89342dd5f9`, private IP `172.31.32.195`, subnet `subnet-0f213d036a08fd543`, SG `sg-07f0487f1de24caed`, AZ `ap-south-1a`.
- Dedicated Lambda SG: `sg-057c9893359ab2300`.
- Existing TCP `29090` remains reserved for monitoring; wallet proxy traffic uses only TCP `29100`.

---

### Task 1: Finalize Seed-Private Nginx Policy

**Files:**
- Modify: `deployment/testnet/public-rpc/nginx-wallet-proxy.conf`
- Modify: `deployment/testnet/public-rpc/nginx_wallet_proxy_test.go`

**Interfaces:**
- Consumes: node RPC on `127.0.0.1:28545`.
- Produces: private wallet-safe HTTP listener semantics for port 29100.

- [ ] **Step 1: Extend the failing policy test** to assert exact methods, no catch-all upstream proxy, body limit on transaction POST, bounded connect/read/send timeouts, and no metrics/blocks/mempool exposure.
- [ ] **Step 2: Run `go test ./deployment/testnet/public-rpc -count=1`** and confirm RED for the new assertions.
- [ ] **Step 3: Make the minimal Nginx template changes** while retaining a default 404 and localhost upstream.
- [ ] **Step 4: Run focused and full Go CI** and confirm GREEN.
- [ ] **Step 5: Compare the repository template against the two already-deployed seed-specific configs** and record any harmless live differences such as private-IP binding and duplicate no-store headers.

### Task 2: Implement Lambda Route Allowlist and Event Normalization

**Files:**
- Create: `deployment/testnet/public-rpc/lambda/package.json`
- Create: `deployment/testnet/public-rpc/lambda/router.mjs`
- Create: `deployment/testnet/public-rpc/lambda/router.test.mjs`
- Create: `deployment/testnet/public-rpc/lambda/index.mjs`

**Interfaces:**
- Consumes: API Gateway HTTP API v2 event with `requestContext.http.method`, `rawPath`, headers, body, and `isBase64Encoded`.
- Produces: normalized `{ method, path, body, headers }` request or a local 4xx response before any upstream network call.

- [ ] **Step 1: Write failing Node tests** covering exactly the six allowed route shapes and rejecting `/metrics`, `/v1/blocks/*`, `/v1/mempool`, malformed account/transaction IDs, unsupported methods, encoded path traversal, and oversized bodies.
- [ ] **Step 2: Run `node --test deployment/testnet/public-rpc/lambda/*.test.mjs`** and confirm RED.
- [ ] **Step 3: Implement minimal pure routing/normalization functions** with no AWS SDK dependency and no transaction-body logging.
- [ ] **Step 4: Run Node tests and secret guard** and confirm GREEN.
- [ ] **Step 5: Commit the route boundary independently.**

### Task 3: Implement Two-Seed Failover and Safe Transaction Retry

**Files:**
- Create: `deployment/testnet/public-rpc/lambda/upstream.mjs`
- Create: `deployment/testnet/public-rpc/lambda/upstream.test.mjs`
- Modify: `deployment/testnet/public-rpc/lambda/index.mjs`

**Interfaces:**
- Consumes: normalized request and configured seed endpoints `http://172.31.10.171:29100`, `http://172.31.32.195:29100`.
- Produces: one authoritative proxied HTTP response, or a truthful 502/503/504 outcome when no authoritative result exists.

- [ ] **Step 1: Write failing tests** for read failover on connection error/timeout/retryable 5xx, no failover on authoritative 4xx application responses, and exact-body transaction retry.
- [ ] **Step 2: Add a test proving POST retry reuses byte-for-byte identical body** and does not mutate transaction ID/body.
- [ ] **Step 3: Implement bounded `fetch` requests with `AbortController`** and deterministic seed order with one failover attempt.
- [ ] **Step 4: Implement safe response filtering** with `Cache-Control: no-store`, content type, request correlation metadata that does not expose secrets, and no upstream `Server` header forwarding.
- [ ] **Step 5: Run Node tests repeatedly and confirm deterministic GREEN.**

### Task 4: Add Deployment Configuration, Monitoring, and Least-Privilege IAM

**Files:**
- Create: `deployment/testnet/public-rpc/aws/README.md`
- Create: `deployment/testnet/public-rpc/aws/lambda-execution-policy.json`
- Create: `deployment/testnet/public-rpc/aws/github-actions-testnet-policy.json`
- Create: `deployment/testnet/public-rpc/aws/http-api-config.md`
- Create: `deployment/testnet/public-rpc/aws/alarms.md`
- Create/Modify: `.github/workflows/testnet-public-rpc.yml`

**Interfaces:**
- Consumes: exact Sudharma Testnet resource IDs and the existing OIDC role.
- Produces: least-privilege deployment/documentation and CI package artifact.

- [ ] **Step 1: Write policy validation tests/guard checks** that reject wildcard admin policies, permanent credentials, or raw 28545 exposure.
- [ ] **Step 2: Define the Lambda execution policy** only for VPC ENI/logging permissions actually required by the function.
- [ ] **Step 3: Define the GitHub OIDC deployment policy** restricted to the exact Lambda/API Gateway/log/alarms resources used by Sudharma Testnet.
- [ ] **Step 4: Document API Gateway throttling, request limits, timeout expectations, access logging, and CloudWatch alarms** for elevated 5xx, Lambda errors/throttles, and latency.
- [ ] **Step 5: Package Lambda from the tested source and publish a CI artifact** without secrets.

### Task 5: Deploy Lambda and API Gateway, Then Smoke-Test Public HTTPS

**Files:**
- Create: `scripts/smoke-public-wallet-api.sh`
- Create: `deployment/testnet/public-rpc/aws/deployment-state.md` after successful deployment.

**Interfaces:**
- Consumes: tested Lambda artifact, VPC subnets/security group, Lambda execution role, API Gateway HTTP API.
- Produces: stable AWS-generated HTTPS `execute-api` endpoint.

- [ ] **Step 1: Deploy the tested Lambda package** to the existing `Sudharma-Testnet-Wallet-Proxy` function using Node.js 24.x and the existing VPC/execution-role configuration.
- [ ] **Step 2: Configure non-secret environment values** for the two private seed endpoints and bounded timeout/body-size settings.
- [ ] **Step 3: Create/configure the HTTP API and `$default` stage** with only the proxy integration routes needed by the six-path allowlist; Lambda still enforces the definitive route boundary.
- [ ] **Step 4: Configure throttling/logging/alarms** and verify Lambda security-group egress plus seed security-group ingress on 29100.
- [ ] **Step 5: Smoke-test `/health`, `/ready`, `/v1/status`, a syntactically valid account lookup, forbidden `/metrics`, and malformed routes from the public HTTPS endpoint.**
- [ ] **Step 6: Exercise failover by making one seed unavailable to Lambda without stopping both seeds, then restore it; verify truthful degraded behavior and no false success.**
- [ ] **Step 7: Record the stable HTTPS base URL and deployment IDs that are safe to commit.**

### Task 6: Add Android Compiled Testnet Endpoint and Preference Migration

**Files:**
- Modify: `mobile/android/app/build.gradle.kts`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletPreferences.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/chain/sudharma/SudharmaRpcClient.kt`
- Test: existing/new tests under `mobile/android/app/src/test/java/network/sudharma/wallet/`

**Interfaces:**
- Consumes: stable public HTTPS base URL from Task 5.
- Produces: Testnet default RPC endpoint with blank-config migration and optional debug-only override.

- [ ] **Step 1: Write failing tests** for fresh install default, blank-value migration, preserved explicit debug override, and no mainnet endpoint.
- [ ] **Step 2: Add a Testnet build constant** for the HTTPS base URL and ensure release/Testnet UI never requires manual RPC entry.
- [ ] **Step 3: Implement migration logic** that replaces only blank/unconfigured legacy values.
- [ ] **Step 4: Run Android unit tests and lint.**

### Task 7: Add Autosync Connection State and Production Portfolio Flow

**Files:**
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/PortfolioState.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/SudharmaWalletRepository.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt`
- Add focused connection-state tests.

**Interfaces:**
- Produces: `Connecting`, `Synchronized(height,lastSync)`, `Degraded(height,lastSync)`, and `Offline(lastSync)` states with bounded exponential backoff.

- [ ] **Step 1: Write failing state-transition/backoff tests.**
- [ ] **Step 2: Route production balance/status refresh through `PortfolioState` and repository state.**
- [ ] **Step 3: Implement bounded backoff and truthful stale/offline presentation without claiming transaction success.**
- [ ] **Step 4: Remove ordinary-user RPC URL form from production UI while retaining a debug-only override path if already supported.**
- [ ] **Step 5: Run unit tests and lint.**

### Task 8: Resolve Payment URI, Prepared Transfer, Recreation, and RPC-ID Release Blockers

**Files:**
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/chain/sudharma/SudharmaChainAdapter.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/chain/sudharma/SudharmaPaymentUri.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/chain/sudharma/SudharmaRpcClient.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/SudharmaWalletRepository.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt`
- Add/modify focused tests.

**Interfaces:**
- Produces: immutable prepared transfer containing recipient, amount, fee, nonce, network, and deterministic unsigned/signing payload before authorization.

- [ ] **Step 1: Write failing tests proving scanned URI amount is preserved and parsing occurs through the chain adapter.**
- [ ] **Step 2: Write failing tests proving fee/nonce are fetched before authorization and confirmation data cannot change afterward.**
- [ ] **Step 3: Write recreation tests proving a prepared/signed/submitting transfer survives state restoration without automatic second signing or ambiguous retry.**
- [ ] **Step 4: Add tests rejecting mismatched RPC response IDs and malformed transaction-status responses.**
- [ ] **Step 5: Implement the minimal production changes and run Android tests/lint.**

### Task 9: Resolve Lifecycle Relock, FLAG_SECURE, Onboarding Recreation, and Cloud Backup Production Use

**Files:**
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/MainActivity.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletFlow.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt`
- Modify production integration points for `mobile/android/app/src/main/java/network/sudharma/wallet/backup/CloudBackupBoundary.kt`
- Add instrumentation tests under `mobile/android/app/src/androidTest/`

**Interfaces:**
- Produces: wallet relock on background, secure-window behavior, recreation-safe onboarding state, and encrypted-only production backup boundary.

- [ ] **Step 1: Write lifecycle/instrumentation tests** for background relock and `FLAG_SECURE` on sensitive screens.
- [ ] **Step 2: Write unit tests for onboarding recreation** that exclude recovery phrase/private material from ordinary saved navigation state.
- [ ] **Step 3: Wire `CloudBackupBoundary` into the optional production backup/export flow** so only encrypted ciphertext crosses the boundary.
- [ ] **Step 4: Implement lifecycle relock and saved-state changes, then run unit/instrumentation tests where supported.**

### Task 10: Integrate the Official Sudharma Logo

**Files:**
- Add authoritative source under Android resources/assets without altering the master source semantics.
- Add generated launcher/adaptive icon derivatives under Android resource directories.
- Modify `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt`, Android theme/resources, and manifest icon references.
- Add asset/config tests.

**Interfaces:**
- Consumes: official 1254x1254 RGBA circular black-and-gold Sudharma logo supplied by the owner.
- Produces: full-logo splash/welcome/About usage, compact header branding, and center-emblem launcher/adaptive icon derivative with clear TESTNET labeling.

- [ ] **Step 1: Verify the supplied official asset against repository `assets/sudharma-logo.png` and choose the owner-supplied file as authoritative when they differ.**
- [ ] **Step 2: Add tests for required resource presence and launcher/adaptive icon configuration.**
- [ ] **Step 3: Generate size-appropriate derivatives from the authoritative source without redesigning the logo.**
- [ ] **Step 4: Integrate full and compact logo placements and preserve TESTNET branding.**
- [ ] **Step 5: Build and visually inspect key screens/launcher assets.**

### Task 11: Full Verification and APK Artifact

**Files:**
- CI/workflow files as needed for reproducible Android artifact generation.

**Interfaces:**
- Produces: installable Testnet debug APK plus SHA-256 and verification report.

- [ ] **Step 1: Run `go test ./... -count=1` and the repository secret guard.**
- [ ] **Step 2: Run all Lambda Node tests.**
- [ ] **Step 3: Run Android unit tests and lint.**
- [ ] **Step 4: Run instrumentation/UI tests where CI supports an emulator; otherwise record the exact owner-device checks still required.**
- [ ] **Step 5: Build the Testnet debug APK and publish/download the workflow artifact.**
- [ ] **Step 6: Compute SHA-256 and verify no credentials, mnemonic, private key, signing keystore, or secret file is packaged/tracked.**
- [ ] **Step 7: Visually inspect the APK/screenshots for logo scaling, TESTNET branding, connection state, and absence of ordinary RPC configuration UI.**
- [ ] **Step 8: Provide the APK to the owner for OnePlus 11R installation and real-device smoke tests.**
- [ ] **Step 9: Keep PR #20 draft/unmerged and label the APK intermediate until independent review closes every release gate.**
