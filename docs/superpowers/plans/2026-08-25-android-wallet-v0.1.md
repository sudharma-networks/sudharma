# Sudharma Wallet Android v0.1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a tested Android debug APK for a non-custodial Sudharma Testnet wallet with a multi-chain-ready architecture.

**Architecture:** A Kotlin Android project under `mobile/android` separates UI/security/recovery from a chain adapter API. The Sudharma adapter reproduces the existing Go P-256/address/transaction/RPC contract and is proven by cross-language golden vectors before Send is enabled.

**Tech Stack:** Kotlin, Android Gradle Plugin, Jetpack Compose, Navigation Compose, Android Keystore/BiometricPrompt, Kotlin coroutines, OkHttp/Kotlin serialization, QR library, JUnit, MockWebServer, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-08-25-android-wallet-v0.1.md`

## Global Constraints
- Sudharma Testnet only in v0.1; Mainnet unavailable.
- Non-custodial local signing; secrets never sent to RPC/Google.
- 12-word BIP39 default; versioned Sudharma mobile derivation profile.
- Preserve exact Go P-256/address/transaction/signature compatibility.
- Optional Google identity/cloud backup cannot block core wallet operation.
- No real swap, BTC/EVM/Solana, staking, dApps or fiat in v0.1.
- Official logo must not be fabricated; use temporary brand mark until supplied.
- Never commit mnemonics, private keys, OAuth secrets, signing keystores or APK binaries.
- Merge only after complete required verification.

---

### Task 1: Android build skeleton and CI
**Files:** Create `mobile/android/settings.gradle.kts`, root Gradle files, `mobile/android/app/*`, `.github/workflows/android-wallet.yml`; modify `.gitignore`.
**Interfaces:** Produces reproducible `:app:assembleDebug`, `:app:testDebugUnitTest`, `:app:lintDebug` and an APK artifact.
- [ ] Add CI workflow first; verify RED because Android project is absent.
- [ ] Add minimal Kotlin/Compose Android project and package namespace.
- [ ] Add generated/signing/OAuth-local files to `.gitignore`.
- [ ] Verify unit/lint/debug build GREEN in GitHub Actions.

### Task 2: Chain-neutral models and adapter contract
**Files:** Create chain/model Kotlin files and unit tests.
**Interfaces:** `ChainAdapter`, `NetworkId`, `AssetBalance`, `FeeQuote`, `UnsignedTransfer`, `SignedTransfer`, `TransactionState`.
- [ ] Write failing tests for money/network validation semantics.
- [ ] Implement minimal immutable models/interfaces.
- [ ] Verify GREEN.

### Task 3: Sudharma protocol compatibility core
**Files:** Create `chain/sudharma/SudharmaCrypto.kt`, `SudharmaTransaction.kt`, tests and golden vectors; add Go vector verification where needed.
**Interfaces:** Address derivation, fee/tx ID, fixed-width P-256 signatures.
- [x] Add deterministic test-only golden vectors and failing Kotlin tests.
- [x] Implement P-256/address/transaction/signature compatibility.
- [x] Verify Go and Kotlin vectors GREEN.

### Task 4: Recovery and deterministic mobile derivation v1
**Files:** Recovery/derivation Kotlin classes and tests.
**Interfaces:** 12-word BIP39 generation/validation plus `deriveSudharmaV1(seed, accountIndex)`.
- [ ] Add BIP39 and project derivation vectors; verify RED.
- [ ] Implement BIP39 via reviewed dependency and domain-separated rejection-sampling derivation.
- [ ] Verify deterministic recovery GREEN.

### Task 5: Secure vault, PIN and biometric gate
**Files:** Security Kotlin classes/tests.
**Interfaces:** Encrypted wallet vault, slow-KDF PIN verifier with backoff, biometric authorization, FLAG_SECURE helper.
- [ ] Add failing persistence/PIN security tests.
- [ ] Implement Keystore-wrapped authenticated encryption and gates.
- [ ] Verify tests/lint GREEN.

### Task 6: Sudharma RPC adapter
**Files:** RPC client/adapter plus MockWebServer fixtures/tests.
**Interfaces:** Status, account balance/nonce, fee, submit signed transaction, transaction status.
- [ ] Write failing request/response/error/timeout tests.
- [ ] Implement strict RPC client with bounded responses/timeouts.
- [ ] Verify no secret material exists in request models and tests GREEN.

### Task 7: Onboarding and secure unlock UI
**Files:** Compose screens/navigation/view models for splash, welcome, create/import, phrase verification, PIN, biometrics, unlock.
- [ ] Write failing state-machine tests.
- [ ] Implement short reduced-motion-aware branded splash and simple onboarding.
- [ ] Protect sensitive screens and verify build/lint/tests GREEN.

### Task 8: Portfolio, network selector and activity UI
**Files:** Portfolio/asset/activity/settings screens and view models.
- [ ] Write failing loading/offline/balance/activity/testnet-label tests.
- [ ] Implement multi-asset-ready portfolio with SUDH only.
- [ ] Keep Mainnet unavailable and Swap `Coming later`.
- [ ] Verify GREEN.

### Task 9: Receive and QR
**Files:** Receive screen, payment URI parser/encoder, QR renderer/scanner.
- [ ] Write failing parser/network/address tests.
- [ ] Implement QR/copy/share/scan permission flow.
- [ ] Verify GREEN.

### Task 10: Send, local signing and transaction status
**Files:** Send/confirmation/status screens/view model/tests.
- [ ] Write failing validation/auth/RPC/status tests.
- [ ] Implement confirm -> authorize -> local sign -> submit -> authoritative status flow.
- [ ] Verify golden compatibility plus unit/UI tests GREEN.

### Task 11: Optional Google identity/cloud-backup boundary
**Files:** Identity/backup feature boundary and tests.
- [ ] Write failing non-blocking/unconfigured and ciphertext-only tests.
- [ ] Implement optional configuration-gated integration without repository secrets.
- [ ] Verify GREEN.

### Task 12: Release verification and APK artifact
**Files:** Docs/workflow updates only as needed.
- [ ] Run existing Go CI and secret guard.
- [ ] Run Android unit tests and lint.
- [ ] Build debug APK and upload workflow artifact.
- [ ] Verify no secrets/APK/keystores are tracked.
- [ ] Open/finalize PR; merge only if every required gate passes, otherwise leave open with exact status.
