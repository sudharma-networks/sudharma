# Android Wallet 0.1.6 Regression Restoration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore the last known-good Android wallet navigation and transaction-history experience on current `main`, publish only `wallet-testnet-0.1.6`, and prevent later releases from silently removing these behaviors.

**Architecture:** Port the proven behavior from `wallet-testnet-0.1.4` onto current `origin/main` without replacing current infrastructure, branding, Gradle wrapper, RPC configuration, or newer changes wholesale. Express navigation and explorer behavior as small pure Kotlin policies with unit tests, wire those policies into Compose, and add a release-contract test plus CI execution so future wallet releases must preserve the user-visible baseline.

**Tech Stack:** Kotlin, Jetpack Compose, JUnit 4, Gradle/Android Gradle Plugin, GitHub Actions, GitHub Releases.

**Spec:** User-approved design in the 2026-09-01 conversation: preserve system Back navigation, separate Activity and History, expose transaction details and blockchain explorer links, release `0.1.6-testnet`, make the features permanent on `main`, and remove all older Android-wallet releases only after verification.

## Global Constraints

- Android application version must become `versionCode = 6` and `versionName = "0.1.6-testnet"`.
- Preserve all current `origin/main` changes unless a tested wallet behavior requires a focused modification.
- System Back must navigate nested wallet screens before Android is allowed to minimize/exit the app.
- Server Activity and wallet transaction History must remain distinct destinations.
- Transaction History must expose transaction details, copyable transaction IDs, and a public explorer action.
- Release automation must run the wallet regression suite before an Android release can be published.
- Delete old Android-wallet releases and matching tags only after `wallet-testnet-0.1.6` assets and checksum are verified.

---

### Task 1: Establish a Clean Current-Main Baseline

**Files:**
- Inspect: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt`
- Inspect: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletFlow.kt`
- Inspect: `.github/workflows/`

**Interfaces:**
- Consumes: `origin/main` at the start of the worktree.
- Produces: recorded baseline test/build evidence and a focused source comparison with tag `wallet-testnet-0.1.4`.

- [ ] **Step 1:** Run `./gradlew test` from `mobile/android` and record the exact result.
- [ ] **Step 2:** Compare current wallet navigation/history files with `wallet-testnet-0.1.4`, preserving current-main files unrelated to the regression.
- [ ] **Step 3:** Confirm the root-cause hypothesis: canonical integration commit `14d15af` replaced the working `f772137`/`31e6d29` wallet UI path.

### Task 2: Restore and Lock System Back Navigation

**Files:**
- Create: `mobile/android/app/src/test/java/network/sudharma/wallet/SystemBackNavigationTest.kt`
- Create: `mobile/android/app/src/main/java/network/sudharma/wallet/SystemBackNavigation.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt`

**Interfaces:**
- Consumes: `WalletScreen` destinations.
- Produces: `SystemBackNavigation.intercepts(WalletScreen): Boolean` and `SystemBackNavigation.previous(WalletScreen): WalletScreen`.

- [ ] **Step 1:** Add table-driven tests proving HOME is not intercepted, top-level child screens return HOME, BACKUP returns SETTINGS, and onboarding screens return their correct parent.
- [ ] **Step 2:** Run `./gradlew test --tests network.sudharma.wallet.SystemBackNavigationTest` and verify the test fails because the policy does not exist.
- [ ] **Step 3:** Add the minimal pure navigation policy and Compose `BackHandler` wiring.
- [ ] **Step 4:** Re-run the focused test and the complete Android unit-test suite; require zero failures.
- [ ] **Step 5:** Commit the tested navigation restoration.

### Task 3: Restore Separate Activity and History Destinations

**Files:**
- Create: `mobile/android/app/src/test/java/network/sudharma/wallet/ActivityHistorySplitTest.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletFlow.kt`
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt`
- Modify as required by the proven implementation: `mobile/android/app/src/main/java/network/sudharma/wallet/SudharmaWalletRepository.kt`

**Interfaces:**
- Consumes: existing server status and transaction activity repository methods.
- Produces: distinct `WalletScreen.ACTIVITY` and `WalletScreen.HISTORY` routes, with Activity rendering server state and History rendering wallet transactions.

- [ ] **Step 1:** Add behavior tests proving Activity and History are distinct routes and both return to HOME.
- [ ] **Step 2:** Run the focused test and verify an expected failure caused by the missing HISTORY destination.
- [ ] **Step 3:** Port the smallest compatible split from commit `31e6d29`, adapting it to current-main APIs instead of replacing current files wholesale.
- [ ] **Step 4:** Re-run focused and full Android tests; require zero failures.
- [ ] **Step 5:** Commit the tested Activity/History split.

### Task 4: Restore Transaction Details and Explorer Actions

**Files:**
- Create: `mobile/android/app/src/test/java/network/sudharma/wallet/ExplorerLinksTest.kt`
- Create: `mobile/android/app/src/test/java/network/sudharma/wallet/TransactionDetailPresentationTest.kt`
- Create or restore focused production files under `mobile/android/app/src/main/java/network/sudharma/wallet/` for explorer links and transaction-detail presentation.
- Modify: `mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt`

**Interfaces:**
- Consumes: transaction ID and wallet transaction record.
- Produces: validated HTTPS transaction URL, visible transaction detail fields, clipboard action, explorer action, and nested-detail Back behavior.

- [ ] **Step 1:** Add tests with hand-derived expected explorer URLs and transaction-detail values.
- [ ] **Step 2:** Run focused tests and verify expected failures because explorer/detail behavior is absent.
- [ ] **Step 3:** Port the minimal compatible behavior from the `0.1.4` implementation, keeping Android intent/clipboard side effects at the UI boundary.
- [ ] **Step 4:** Re-run focused and full Android tests; require zero failures.
- [ ] **Step 5:** Commit the tested transaction-detail and explorer restoration.

### Task 5: Add the Permanent Release Regression Gate

**Files:**
- Create: `mobile/android/app/src/test/java/network/sudharma/wallet/WalletReleaseBaselineTest.kt`
- Modify: the Android CI/release workflow selected from `.github/workflows/`.
- Modify: `mobile/android/app/build.gradle.kts`

**Interfaces:**
- Consumes: the public navigation, history, detail, and explorer policies from Tasks 2-4.
- Produces: a release-blocking automated suite and Android version `0.1.6-testnet` / code `6`.

- [ ] **Step 1:** Add an integration-style release baseline test that exercises user-observable navigation contracts without source-text assertions.
- [ ] **Step 2:** Run it before completing the release wiring and verify the expected failure.
- [ ] **Step 3:** Update the version and make the Android release workflow execute the full unit-test task before building/signing.
- [ ] **Step 4:** Run focused tests, full Android tests, lint, and release assembly; require zero failures.
- [ ] **Step 5:** Commit the version and release gate.

### Task 6: Integrate, Publish, Verify, and Retire Old Releases

**Files:**
- Modify if present: website/download metadata pointing to the Android wallet release.
- Produce: signed `0.1.6` APK and SHA-256 checksum through the repository's release process.

**Interfaces:**
- Consumes: fully verified branch and configured repository release credentials.
- Produces: current `main`, tag/release `wallet-testnet-0.1.6`, downloadable APK/checksum, and no older Android-wallet release entries or matching tags.

- [ ] **Step 1:** Verify the full repository-required checks and inspect the complete diff against `origin/main`.
- [ ] **Step 2:** Push a branch and create a PR to `main`; merge only after required checks pass.
- [ ] **Step 3:** Build/publish `wallet-testnet-0.1.6` from the merged commit and verify APK version, SHA-256 asset, and download URL.
- [ ] **Step 4:** Update and verify website/download metadata if it is not generated automatically.
- [ ] **Step 5:** Delete only the older Android-wallet GitHub releases and matching tags (`wallet-testnet-0.1.5`, `wallet-testnet-0.1.4`, and `android-wallet-v0.1.0-testnet`, plus any confirmed Android-only legacy tags), leaving non-wallet releases untouched.
- [ ] **Step 6:** Re-query `main`, tags, releases, and assets to prove only `wallet-testnet-0.1.6` remains for Android.
