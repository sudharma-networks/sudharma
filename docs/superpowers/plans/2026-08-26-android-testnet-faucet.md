# Sudharma Android Testnet Faucet Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a Sudharma Android testnet wallet with the permanent public RPC configured by default and a safe one-time **Get 100 Test SUDH** faucet backed by a dedicated server-side faucet wallet.

**Architecture:** Keep the existing public wallet RPC Lambda unchanged as the general wallet proxy and add a separate faucet service for fixed-amount grants. The faucet service validates one 40-hex address, reserves the address atomically in DynamoDB, signs a fixed 100 SUDH transaction using a dedicated faucet wallet secret held in AWS Secrets Manager, submits through the trusted Sudharma node RPC path, and records the result. Android calls only `POST /v1/faucet/request`, never receives or transmits faucet private-key material, and keeps the existing manual RPC override while defaulting fresh installs to the proven public HTTPS endpoint.

**Tech Stack:** Android/Kotlin 17, Jetpack Compose, OkHttp/Moshi, Node.js 24 Lambda, AWS API Gateway HTTP API, AWS DynamoDB, AWS Secrets Manager/KMS, GitHub Actions OIDC, existing Sudharma RPC protocol.

**Spec:** `docs/superpowers/specs/2026-08-26-android-testnet-faucet-design.md`

## Global Constraints

- Public Android RPC default is exactly `https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com`.
- Faucet grant is exactly `10000000000` atomic units = 100 SUDH.
- Initial eligibility is one successful faucet grant per normalized wallet address.
- No GitHub account, email, login, device ID, analytics ID, or recovery phrase is required.
- Faucet client sends only the public Sudharma address.
- Faucet private key must never be committed, logged, embedded in APK resources, or exposed through API responses.
- Public faucet requests must not trigger mining.
- Public faucet API must not accept arbitrary amount or source-wallet parameters.
- Existing Android Send/Receive flow and local signing remain unchanged.
- Existing public wallet RPC proxy stays a distinct deployment and security boundary.

---

## File Structure

### Android
- Modify `mobile/android/app/src/main/java/network/sudharma/wallet/WalletPreferences.kt` — permanent RPC default while preserving override.
- Create `mobile/android/app/src/main/java/network/sudharma/wallet/faucet/FaucetClient.kt` — HTTP request/response boundary for faucet only.
- Create `mobile/android/app/src/main/java/network/sudharma/wallet/faucet/FaucetModels.kt` — request/result/state models.
- Modify `mobile/android/app/src/main/java/network/sudharma/wallet/SudharmaWalletRepository.kt` — expose `requestTestFunds()` using account public address only.
- Modify `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt` — add button and request state to Home screen.
- Create `mobile/android/app/src/test/java/network/sudharma/wallet/WalletPreferencesTest.kt`.
- Create `mobile/android/app/src/test/java/network/sudharma/wallet/faucet/FaucetClientTest.kt`.
- Create `mobile/android/app/src/test/java/network/sudharma/wallet/faucet/FaucetPrivacyTest.kt`.

### Faucet backend
- Create `deployment/testnet/faucet/lambda/package.json`.
- Create `deployment/testnet/faucet/lambda/index.mjs` — API Gateway handler and safe logging.
- Create `deployment/testnet/faucet/lambda/request.mjs` — strict request parsing/address validation.
- Create `deployment/testnet/faucet/lambda/grants.mjs` — DynamoDB reservation/update contract.
- Create `deployment/testnet/faucet/lambda/wallet.mjs` — faucet secret loading, Sudharma transaction construction/signing, node submission.
- Create matching `*.test.mjs` files for each module.
- Create `deployment/testnet/faucet/aws-resources.json` — non-secret resource names/ARN placeholders resolved from environment at deployment time; no credentials.
- Create `.github/workflows/testnet-faucet.yml` — test/package/deploy Lambda with GitHub OIDC.

### Documentation
- Create `docs/testnet/android-public-testing.md` — public test instructions and safety note.
- Modify `.github/workflows/android-wallet.yml` — include the new spec/plan/docs paths where appropriate.

---

### Task 1: Lock the Android RPC default

**Files:**
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletPreferences.kt`
- Create: `mobile/android/app/src/test/java/network/sudharma/wallet/WalletPreferencesTest.kt`

**Interfaces:**
- Produces: `WalletPreferences.DEFAULT_TESTNET_RPC: String`
- Existing `rpcUrl` setter remains the manual override mechanism.

- [ ] **Step 1: Write the failing test**

```kotlin
@Test
fun freshPreferencesUsePublicTestnetRpc() {
    assertEquals(
        "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com",
        WalletPreferences.DEFAULT_TESTNET_RPC,
    )
}
```

Add a second test asserting an explicitly stored override is returned unchanged except for trailing slash trimming.

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
cd mobile/android
gradle --no-daemon :app:testDebugUnitTest --tests network.sudharma.wallet.WalletPreferencesTest
```

Expected: FAIL because `DEFAULT_TESTNET_RPC` does not yet exist and the current getter defaults to `""`.

- [ ] **Step 3: Implement the minimal production change**

Add:

```kotlin
companion object {
    const val DEFAULT_TESTNET_RPC = "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com"
    // keep validateRpcUrl here as today
}
```

Change only the getter default:

```kotlin
get() = prefs.getString("testnet_rpc_url", DEFAULT_TESTNET_RPC) ?: DEFAULT_TESTNET_RPC
```

Keep the existing setter and HTTPS validation behavior.

- [ ] **Step 4: Run test, lint, and Android build**

```bash
cd mobile/android
gradle --no-daemon :app:testDebugUnitTest
gradle --no-daemon :app:lintDebug
gradle --no-daemon :app:assembleDebug
```

Expected: all commands exit 0.

- [ ] **Step 5: Commit**

```bash
git add mobile/android/app/src/main/java/network/sudharma/wallet/WalletPreferences.kt \
        mobile/android/app/src/test/java/network/sudharma/wallet/WalletPreferencesTest.kt
git commit -m "feat(android): default wallet to public testnet RPC"
```

---

### Task 2: Prove ordinary wallet transfer before faucet exposure

**Files:**
- No production-code change unless the observed transaction path exposes a bug.
- Existing tests: `mobile/android/app/src/test/java/network/sudharma/wallet/chain/sudharma/SudharmaTransferFlowTest.kt`

**Interfaces:**
- Consumes existing Android `send()` and node `POST /v1/transactions` behavior.
- Produces a recorded validation result used as the release gate for later tasks.

- [ ] **Step 1: Create Phone B wallet** and record only its public receive address.
- [ ] **Step 2: From Phone A, send 1 SUDH to Phone B** using the public RPC.
- [ ] **Step 3: Verify node mempool increases** via `/v1/status` and transaction status endpoint.
- [ ] **Step 4: Mine/confirm the pending transaction through the normal PoW path**; do not bypass mempool or edit state files.
- [ ] **Step 5: Verify Phone A and Phone B balances update correctly** and capture tx id/block hash in the test log.
- [ ] **Step 6: If any defect appears, stop here and fix that defect with its own failing regression test before continuing.**

No faucet deployment begins until this task passes.

---

### Task 3: Implement strict faucet request parsing

**Files:**
- Create: `deployment/testnet/faucet/lambda/package.json`
- Create: `deployment/testnet/faucet/lambda/request.mjs`
- Create: `deployment/testnet/faucet/lambda/request.test.mjs`

**Interfaces:**
- Produces: `parseFaucetRequest(event) -> { address: string }`
- Address grammar: lowercase exactly 40 hexadecimal characters.
- Request body schema: exactly one supported field, `address`; no amount/source/private-key fields.

- [ ] **Step 1: Write failing Node tests** for valid address, uppercase/malformed address, missing body, oversized body, extra `amount` field, and non-POST method.
- [ ] **Step 2: Run tests to verify RED**

```bash
cd deployment/testnet/faucet/lambda
npm test
```

- [ ] **Step 3: Implement parser** with `MAX_REQUEST_BYTES = 4096`, JSON-only POST handling, lowercase-40-hex regex, and deterministic 400/405/413 errors.
- [ ] **Step 4: Run `npm test` and verify PASS**.
- [ ] **Step 5: Commit**

```bash
git add deployment/testnet/faucet/lambda
git commit -m "feat(faucet): validate fixed testnet grant requests"
```

---

### Task 4: Add atomic one-grant-per-address ledger

**Files:**
- Create: `deployment/testnet/faucet/lambda/grants.mjs`
- Create: `deployment/testnet/faucet/lambda/grants.test.mjs`

**Interfaces:**
- Consumes AWS DynamoDB Document-style client injected into functions for tests.
- Produces:

```js
reserveGrant({ address, now })
markSubmitted({ address, txId, now })
markFailed({ address, reason, now })
getGrant(address)
```

- DynamoDB partition key: `address`.
- Reserved record uses `amount_atomic: 10000000000` and `status: "reserved"`.

- [ ] **Step 1: Write failing tests** proving two concurrent reservation attempts for the same address yield exactly one winner.
- [ ] **Step 2: Add tests** for `submitted`, duplicate lookup, and retryable `failed` state semantics defined in the spec.
- [ ] **Step 3: Implement conditional DynamoDB write** using `attribute_not_exists(address)` for the first reservation.
- [ ] **Step 4: Run `npm test`** and verify all grant-ledger tests pass.
- [ ] **Step 5: Commit**

```bash
git add deployment/testnet/faucet/lambda/grants.mjs deployment/testnet/faucet/lambda/grants.test.mjs
git commit -m "feat(faucet): add idempotent grant ledger"
```

---

### Task 5: Implement dedicated faucet-wallet signing and node submission

**Files:**
- Create: `deployment/testnet/faucet/lambda/wallet.mjs`
- Create: `deployment/testnet/faucet/lambda/wallet.test.mjs`

**Interfaces:**
- Consumes environment variables:
  - `FAUCET_SECRET_ID`
  - `NODE_RPC_BASE_URL`
  - `REQUEST_TIMEOUT_MS`
- Secret JSON contains only the dedicated testnet faucet recovery/private-key material required to derive the faucet account; the exact serialized secret shape is documented in deployment notes but never committed with real values.
- Produces:

```js
sendGrant({ toAddress }) -> {
  txId,
  amountAtomic: 10000000000,
  fromAddress
}
```

- [ ] **Step 1: Write failing tests** with injected fake secret provider and fake fetch implementation.
- [ ] **Step 2: Assert fixed amount cannot be overridden** even if caller supplies any extra property.
- [ ] **Step 3: Assert transaction fee is included in faucet-wallet balance sufficiency check and next nonce comes from `/v1/accounts/{faucetAddress}`.**
- [ ] **Step 4: Implement transaction construction/signing matching the Sudharma canonical transaction rules and submit to `POST /v1/transactions`.**
- [ ] **Step 5: Ensure logs/tests never output secret scalar/recovery phrase/signature bytes.**
- [ ] **Step 6: Run `npm test` and commit.**

```bash
git add deployment/testnet/faucet/lambda/wallet.mjs deployment/testnet/faucet/lambda/wallet.test.mjs
git commit -m "feat(faucet): sign fixed grants with dedicated wallet"
```

---

### Task 6: Compose the faucet Lambda handler

**Files:**
- Create: `deployment/testnet/faucet/lambda/index.mjs`
- Create: `deployment/testnet/faucet/lambda/index.test.mjs`

**Interfaces:**
- HTTP endpoint: `POST /v1/faucet/request`
- Success response:

```json
{"status":"submitted","amount_atomic":10000000000,"tx_id":"<64-hex>"}
```

- Duplicate response:

```json
{"status":"already_funded"}
```

- Invalid address = 400; rate limit is enforced by API Gateway; depleted faucet or node unavailable = 503; unexpected internal error = 500.

- [ ] **Step 1: Write failing handler tests** for success, duplicate, invalid request, node failure, and secret-safe logs.
- [ ] **Step 2: Implement flow:** parse -> reserve -> send fixed grant -> mark submitted -> safe JSON response.
- [ ] **Step 3: For a submission failure, mark grant `failed` with a sanitized reason and return 503; do not return a secret or signed raw transaction.**
- [ ] **Step 4: Run `npm test` and verify PASS.**
- [ ] **Step 5: Run tracked-secret check**

```bash
bash ./scripts/check-tracked-secrets_test.sh
```

- [ ] **Step 6: Commit.**

---

### Task 7: Add AWS deployment workflow and least-privilege configuration

**Files:**
- Create: `.github/workflows/testnet-faucet.yml`
- Create: `deployment/testnet/faucet/aws-resources.json`
- Modify only if needed: IAM trust/permissions outside repo through AWS console/policy management; never store credentials in Git.

**Interfaces:**
- Lambda function name: `Sudharma-Testnet-Faucet`
- DynamoDB table name: `Sudharma-Testnet-Faucet-Grants`
- Secrets Manager secret id: `sudharma/testnet/faucet-wallet`
- API route: `POST /v1/faucet/request`
- Region: `ap-south-1`

- [ ] **Step 1: Write workflow to run secret scan and `npm test` before packaging.**
- [ ] **Step 2: Package only runtime files (`index.mjs`, `request.mjs`, `grants.mjs`, `wallet.mjs`, `package.json`).**
- [ ] **Step 3: Configure GitHub OIDC using the existing `Sudharma-GitHub-Actions-Testnet` role, extending its permissions only for the specific faucet Lambda update and non-secret configuration.**
- [ ] **Step 4: Create DynamoDB table with address partition key and on-demand capacity; enable point-in-time recovery if available for the environment.**
- [ ] **Step 5: Create Secrets Manager secret manually with the dedicated faucet wallet material; do not commit the value.**
- [ ] **Step 6: Give `Sudharma-Testnet-Faucet` execution role only `secretsmanager:GetSecretValue` on that one secret, required DynamoDB item operations on that one table, VPC/network logging permissions, and CloudWatch Logs. No mining permission exists.**
- [ ] **Step 7: Add API Gateway route to the faucet Lambda and configure throttling before public enablement.**
- [ ] **Step 8: Verify `/v1/status` still routes to `Sudharma-Testnet-Wallet-Proxy` while `/v1/faucet/request` routes only to `Sudharma-Testnet-Faucet`.**
- [ ] **Step 9: Commit workflow/resource metadata.**

---

### Task 8: Create and pre-fund the dedicated faucet wallet

**Files:**
- No secret material committed.
- Add operational notes only to `docs/testnet/android-public-testing.md` after the procedure is proven.

**Interfaces:**
- Faucet wallet address is public and may be documented.
- Faucet secret remains only in AWS Secrets Manager.

- [ ] **Step 1: Generate the faucet wallet in a controlled admin environment and store the secret in AWS Secrets Manager immediately.**
- [ ] **Step 2: Record only the public faucet address in deployment notes.**
- [ ] **Step 3: Pre-fund it by mining administrator-controlled blocks to that faucet address; public requests must never invoke this mining path.**
- [ ] **Step 4: Verify faucet balance through the normal account RPC endpoint.**
- [ ] **Step 5: Perform one backend-only test grant to a fresh test address and confirm the transaction through normal consensus.**

---

### Task 9: Add Android FaucetClient and privacy boundary

**Files:**
- Create: `mobile/android/app/src/main/java/network/sudharma/wallet/faucet/FaucetModels.kt`
- Create: `mobile/android/app/src/main/java/network/sudharma/wallet/faucet/FaucetClient.kt`
- Create: `mobile/android/app/src/test/java/network/sudharma/wallet/faucet/FaucetClientTest.kt`
- Create: `mobile/android/app/src/test/java/network/sudharma/wallet/faucet/FaucetPrivacyTest.kt`

**Interfaces:**

```kotlin
data class FaucetResult(
    val status: FaucetStatus,
    val amountAtomic: Long? = null,
    val transactionId: String? = null,
)

enum class FaucetStatus { SUBMITTED, ALREADY_FUNDED }

class FaucetClient(private val baseUrl: String, private val httpClient: OkHttpClient = OkHttpClient()) {
    suspend fun request(address: String): FaucetResult
}
```

- [ ] **Step 1: Write MockWebServer failing tests** for submitted, already-funded, 400, 429, 503, malformed JSON, and timeout.
- [ ] **Step 2: Add privacy test asserting serialized request keys equal exactly `setOf("address")`.**
- [ ] **Step 3: Implement client using `POST {baseUrl}/v1/faucet/request` and JSON `{ "address": ... }`.**
- [ ] **Step 4: Run focused Android tests and verify PASS.**
- [ ] **Step 5: Commit.**

---

### Task 10: Wire faucet into repository and Home UI

**Files:**
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/SudharmaWalletRepository.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt`
- Create: `mobile/android/app/src/test/java/network/sudharma/wallet/FaucetPresentationTest.kt`

**Interfaces:**
- Add repository method:

```kotlin
suspend fun requestTestFunds(): FaucetResult
```

It calls `account().address` and must not access/export `privateScalar` for the request.

- [ ] **Step 1: Write failing repository/presentation tests** for idle, requesting, submitted, already-funded, and retryable error states.
- [ ] **Step 2: Implement `requestTestFunds()` using `FaucetClient(preferences.rpcUrl)` with only the public account address.**
- [ ] **Step 3: Add `Get 100 Test SUDH` button to Home screen, disabled during request and after deterministic already-funded response.**
- [ ] **Step 4: On success, show tx id/status and refresh balance; do not claim confirmed balance until the account RPC reports it.**
- [ ] **Step 5: Display user-safe errors for timeout, 429, faucet depleted/503, and already funded.**
- [ ] **Step 6: Run full Android test/lint/build commands and commit.**

---

### Task 11: Public testing documentation and CI coverage

**Files:**
- Create: `docs/testnet/android-public-testing.md`
- Modify: `.github/workflows/android-wallet.yml`

**Interfaces:**
- Public docs must include the permanent RPC URL, explicit `TESTNET` warning, 100 SUDH faucet instructions, Send/Receive test flow, issue-reporting guidance, and a warning never to post recovery phrases/private keys.

- [ ] **Step 1: Write the testing document** with install -> create wallet -> Get 100 Test SUDH -> create second wallet -> Send -> Receive -> confirmation flow.
- [ ] **Step 2: State prominently that test SUDH has no monetary value and may be reset.**
- [ ] **Step 3: Update Android workflow path filters to include this plan/spec/docs and retain unit-test -> lint -> APK -> artifact order.**
- [ ] **Step 4: Run repo secret-scan and Android CI-equivalent commands locally/CI.**
- [ ] **Step 5: Commit.**

---

### Task 12: End-to-end public-testnet release gate

**Files:**
- No production edits unless a defect is found.

- [ ] **Step 1: Install the newly built APK as a fresh install on Wallet A and verify no RPC setup is requested.**
- [ ] **Step 2: Verify `/v1/status` through the built-in RPC and confirm current network height/tip.**
- [ ] **Step 3: Tap `Get 100 Test SUDH`; verify faucet returns one tx id and the grant ledger has one address record.**
- [ ] **Step 4: Confirm Wallet A eventually shows 100 SUDH minus no user-side faucet fee deduction (the faucet source pays the fee).**
- [ ] **Step 5: Tap faucet again and verify deterministic `already_funded` with no second transaction.**
- [ ] **Step 6: Create Wallet B and send a small amount from A to B.**
- [ ] **Step 7: Verify mempool -> mined block -> confirmed transaction -> both balances.**
- [ ] **Step 8: Verify Seed-1/Seed-2 remain synchronized and public `/v1/status` remains healthy.**
- [ ] **Step 9: Verify CloudWatch logs/metrics show request counts, accepted/duplicate/error counts, latency, and tx ids but no secret material.**
- [ ] **Step 10: Only after all checks pass, publish the updated APK artifact/release and public testing instructions.**

---

## Self-review checklist

- Spec coverage: permanent RPC, one-time 100 SUDH, dedicated faucet wallet, no login, no mining per request, secret storage, idempotency, rate limiting, monitoring, Android button, docs, and end-to-end validation are each assigned to tasks above.
- Placeholder scan: no implementation step relies on `TBD`, `TODO`, or unspecified generic error handling.
- Type consistency: Android `FaucetResult`/`FaucetStatus` and backend success/duplicate response names match across Tasks 6, 9, and 10.
- Security consistency: public RPC proxy and faucet Lambda remain separate; only the faucet Lambda reads its dedicated secret; client never selects amount/source.
- Release gate: ordinary send/receive is validated before public faucet exposure, and full faucet + transfer flow is validated before APK publication.
