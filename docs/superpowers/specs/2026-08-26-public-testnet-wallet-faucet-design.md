# Sudharma Public Testnet Wallet and Challenge Design

## Goal
Make the Sudharma Android testnet wallet publicly downloadable without requiring a GitHub account, automatically connect it to the public testnet RPC, and provide a controlled multi-day faucet challenge that generates genuine testnet transactions.

## Public wallet distribution
The Android APK is published as a public GitHub Release asset. Anyone can download the APK without signing in to GitHub. The repository README prominently links to the latest testnet wallet release and labels the software TESTNET / PRE-MAINNET.

The wallet ships with this default RPC base URL:

`https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com`

Users do not need to configure RPC manually. An advanced override may remain for development and recovery.

## Public testnet challenge
Participation does not require a GitHub account.

A tester supplies a valid Sudharma testnet receiving address through a public faucet endpoint/page. The faucet gives that address exactly one initial grant of 100 SUDH.

The tester can then complete up to five challenge rounds. In each round:

1. The tester sends exactly 25 SUDH from their wallet to the published official Sudharma challenge deposit address.
2. The backend verifies the transaction on the Sudharma testnet and verifies that it has not already been claimed.
3. The backend sends 50 SUDH back to the tester's registered address.
4. The next round becomes eligible 24 hours after the previous round was successfully completed.

Ignoring transaction fees, balances progress from 100 SUDH to 125, 150, 175, 200, and 225 SUDH after five completed rounds.

The initial 100 SUDH is paid only once. It is not repeated at the start of later rounds.

## Eligibility and abuse controls
Eligibility is tracked primarily by Sudharma address because GitHub authentication is not required. Each address can receive one initial grant and at most five challenge rewards. Each incoming transaction ID can be claimed only once. A challenge reward is issued only after the backend verifies the incoming 25 SUDH transaction on-chain and confirms the 24-hour cooldown has expired.

The public API must apply request throttling and reasonable anti-automation controls. The backend persists grant state, completed rounds, timestamps, and claimed transaction IDs. No client-supplied balance, round number, or timestamp is trusted.

## Faucet and challenge wallet security
Only public deposit addresses are published. Faucet/challenge private keys are never committed to GitHub, embedded in the APK, returned by an API, or written to public logs. Production testnet signing secrets are stored in AWS Secrets Manager (or an equivalent server-side secret store) and are accessible only to the narrowly scoped faucet backend role.

The faucet has an operational balance ceiling and can be paused independently of the public RPC and wallet download.

## Testnet economics
SUDH distributed by this system is testnet currency with no monetary value. The UI, README, faucet page, and release notes state this clearly.

The faucet must be funded with legitimate Sudharma testnet issuance before public launch. Funding is operationally separate from faucet claims; a claim must never create coins or modify balances directly.

## Components

### Android wallet
- Default RPC points to the API Gateway HTTPS endpoint.
- Existing wallets and private keys survive an APK update.
- Wallet can query status/balance and submit transactions through the same RPC.
- TESTNET labeling is visible.

### GitHub public distribution
- README contains a public Android Wallet section.
- Latest APK is a GitHub Release asset rather than a short-lived Actions artifact.
- Release notes identify network, version, default RPC, checksum, and testnet warning.

### Faucet/challenge backend
- Initial-grant endpoint accepts a Sudharma address and returns an idempotent result.
- Challenge-claim endpoint accepts the registered address and incoming transaction ID.
- Backend independently queries the chain before signing a payout.
- Persistent state enforces one initial grant, five rounds, 24-hour cooldown, and transaction-ID uniqueness.

### Public challenge page
- No GitHub login required.
- Shows official deposit address, rules, current eligibility, and next eligible time.
- Allows initial 100-SUDH request and later 25-SUDH transaction submission.
- Does not expose signing credentials or administrative controls.

## Failure behavior
Duplicate initial requests return the existing grant state rather than paying again. Early round requests return the next eligible timestamp without paying. Invalid, missing, wrong-value, wrong-recipient, or already-claimed transactions are rejected without payout. Backend/RPC failures fail closed: no reward is issued until verification succeeds.

## Verification before public launch
1. API Gateway `/v1/status` works from a non-AWS Internet connection.
2. Fresh APK automatically connects without manual RPC configuration.
3. Existing Phone A wallet survives APK update and reads its on-chain balance.
4. Two-device send/receive succeeds through the public RPC.
5. Faucet grants exactly 100 SUDH once to a fresh address.
6. A verified 25-SUDH deposit triggers exactly one 50-SUDH reward.
7. Immediate replay and duplicate transaction claims are rejected.
8. A second round is rejected before 24 hours and accepted after the cooldown in a controlled test.
9. Sixth reward attempt is rejected.
10. No private keys or secrets appear in repository history, APK resources, CI logs, or public API responses.

## Rollout order
First complete and verify the wallet's default RPC and ordinary transaction path. Second publish the APK through a stable GitHub Release. Third create and fund a dedicated faucet/challenge wallet. Fourth deploy the protected faucet backend and persistence. Fifth expose the no-login challenge page. Finally run abuse, replay, cooldown, and end-to-end tests before advertising the faucet publicly.
