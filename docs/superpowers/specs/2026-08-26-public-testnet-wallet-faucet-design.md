# Sudharma Public Testnet Wallet and Challenge Design

## Goal
Provide a publicly downloadable Android testnet wallet that connects automatically to Sudharma Testnet and a no-login faucet/challenge that creates repeated real testnet transactions.

## Wallet distribution
- Public GitHub Release APK; GitHub login is not required to download.
- Default RPC: `https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com`.
- Existing wallet state must survive APK updates.
- Manual RPC override may remain as an advanced option.
- All public surfaces say TESTNET / PRE-MAINNET and test SUDH has no monetary value.

## Challenge rules
- One initial grant of exactly 100 SUDH per Sudharma address.
- Tester sends exactly 25 SUDH to the official challenge deposit address.
- After independent on-chain verification, backend returns exactly 50 SUDH.
- Maximum five successful challenge rounds per address.
- Each later round is eligible 24 hours after the previous successful round.
- Initial 100 SUDH is not repeated in rounds 2–5.
- Ignoring fees, balances progress 100 → 125 → 150 → 175 → 200 → 225 SUDH.
- GitHub account is not required.

## Abuse and security controls
- Track address, initial grant, completed rounds, last-success time, payout txids, and claimed incoming txids in persistent server-side state.
- Reject duplicate grants, duplicate txids, wrong amount, wrong recipient, unconfirmed transactions, early cooldown requests, and round 6+.
- Do not trust client-supplied balances, times, or round numbers.
- Apply throttling and anti-automation controls to public endpoints.
- Faucet/challenge private keys must never be in GitHub, APK, CI logs, or API responses; store signing material in AWS Secrets Manager and use a least-privilege backend role.
- Fail closed when RPC or verification is unavailable.

## Components
1. Android wallet default-RPC behavior and transaction verification.
2. GitHub Release workflow and README public download section.
3. Dedicated faucet/challenge wallet funded by legitimate testnet issuance/transfers.
4. AWS-backed faucet eligibility state, chain verifier, and signer.
5. Public no-login challenge page/API.

## Rollout order
1. Default RPC APK and existing-wallet update test.
2. End-to-end Phone A → Phone B send/receive through public RPC.
3. Stable public APK release.
4. Dedicated faucet wallet creation/funding and secret storage.
5. Faucet/challenge backend with idempotency, cooldown, and replay protection.
6. Public no-login challenge interface.
7. Abuse/security/end-to-end verification before public announcement.
