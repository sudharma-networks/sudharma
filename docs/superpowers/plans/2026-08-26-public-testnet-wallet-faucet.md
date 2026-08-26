# Sudharma Public Testnet Wallet and Challenge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a public Android testnet wallet with automatic RPC plus a no-login, abuse-controlled 100-SUDH faucet and five-round 25→50 SUDH challenge.

**Architecture:** Android uses the existing API Gateway endpoint as its default RPC. GitHub Releases provides stable APK distribution. A separate AWS backend verifies chain activity, persists eligibility, and signs faucet/reward payouts using server-side secrets.

**Tech Stack:** Kotlin/Android, Gradle, GitHub Actions/Releases, Go Sudharma RPC, AWS API Gateway/Lambda, AWS Secrets Manager, persistent AWS datastore.

**Spec:** `docs/superpowers/specs/2026-08-26-public-testnet-wallet-faucet-design.md`

## Global Constraints
- Default RPC: `https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com`
- Initial grant: 100 SUDH once per address.
- Challenge: exactly 25 SUDH in, 50 SUDH reward out.
- Maximum five rounds; 24-hour cooldown between successful rounds.
- No GitHub account required for APK download or challenge participation.
- No private key in source, APK, CI logs, or public responses.
- TESTNET / no monetary value warnings on public surfaces.

---

### Task 1: Default RPC in Android wallet
**Files:** modify `mobile/android/app/src/main/java/network/sudharma/wallet/WalletPreferences.kt`; create/update focused tests under `mobile/android/app/src/test/`.

- [ ] Write failing test for the exact default RPC constant/value.
- [ ] Run Android CI/unit test and verify RED failure is caused by missing default.
- [ ] Implement minimal default while preserving explicit override and URL validation.
- [ ] Run unit tests, lint, and debug APK build; verify GREEN.
- [ ] Install update over existing wallet and verify wallet address survives and balance is queryable.

### Task 2: Verify ordinary transaction path
- [ ] Confirm Phone A balance through public RPC.
- [ ] Create/use Phone B address.
- [ ] Send a small transaction Phone A → Phone B.
- [ ] Verify mempool, block inclusion, and balances on both seeds.
- [ ] Fix/test any discovered RPC or Android defect before faucet work.

### Task 3: Public APK release and README
**Files:** `.github/workflows/android-wallet.yml`, `README.md`.

- [ ] Add version-tag release build and SHA-256 generation.
- [ ] Publish APK as a GitHub Release asset using repository-scoped token only.
- [ ] Add public Android Wallet section with release/download link, default RPC statement, and TESTNET warning.
- [ ] Verify logged-out download works.

### Task 4: Dedicated faucet/challenge wallet
- [ ] Generate a dedicated testnet wallet with project wallet code.
- [ ] Publish only its address.
- [ ] Store signing secret in AWS Secrets Manager with least privilege.
- [ ] Fund it through legitimate mining/transfers and verify balance through RPC.
- [ ] Document refill/pause/recovery without recording secrets.

### Task 5: Persistent faucet eligibility
- [ ] Write tests for one-time 100-SUDH grant.
- [ ] Write tests for rounds 1–5, round 6 rejection, 24-hour cooldown, and txid replay rejection.
- [ ] Implement atomic persistent state so concurrent requests cannot duplicate payouts.
- [ ] Run concurrency/idempotency tests.

### Task 6: On-chain verifier and payout signer
- [ ] Write tests for wrong recipient, wrong amount, unconfirmed/unknown/reused txid.
- [ ] Verify exact 25-SUDH confirmed transfer to official challenge address from chain data.
- [ ] Load signing secret server-side and send exactly 100 SUDH initial or 50 SUDH challenge payout.
- [ ] Make payout processing idempotent around broadcast failures/retries.
- [ ] Ensure no secrets are logged or returned.

### Task 7: Public no-login challenge API/page
- [ ] Add initial-grant request accepting a valid Sudharma address.
- [ ] Add challenge-claim request accepting address + incoming txid.
- [ ] Return current round, next eligible time, and payout txid/status without exposing secrets.
- [ ] Add throttling/anti-automation controls and TESTNET warning.
- [ ] Verify GitHub account is never required.

### Task 8: Public-launch verification
- [ ] Fresh APK auto-connects to RPC.
- [ ] Existing Phone A wallet survives update.
- [ ] Phone A → Phone B transaction succeeds.
- [ ] Fresh address gets 100 exactly once.
- [ ] Confirmed 25 triggers one 50 reward.
- [ ] Duplicate txid and early cooldown are rejected.
- [ ] Five rounds accepted over controlled time tests; sixth rejected.
- [ ] Repository/APK/log scan finds no faucet private key or secret.
