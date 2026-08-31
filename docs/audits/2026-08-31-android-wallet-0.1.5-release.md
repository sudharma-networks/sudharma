# Android wallet 0.1.5 go-live release

**Recorded:** 2026-08-31  
**Release tag:** `wallet-testnet-0.1.5`  
**Release URL:** https://github.com/sudharma-networks/sudharma/releases/tag/wallet-testnet-0.1.5  
**Source commit:** `dfe5e740237202ec6d261ef862b15bdc7e9a05db`  
**APK:** `Sudharma-Wallet-0.1.5.apk`  
**SHA-256:** `486c0c233a4eb53b3292d643082e936c0599804063ffd15290f0edd2b50f9956`

## Outcome

Stage 5 step 7 (Android APK release) is **complete**.

| Step | Result |
| --- | --- |
| Version bump + publish workflow | Merged in PR #73 |
| Android CI build on `main` | Run `33416152288` success |
| GitHub prerelease | Published `wallet-testnet-0.1.5` |
| Website release sync | Run `33416400499` success (release-triggered) |
| Amplify downloads | Live `wallet-testnet-0.1.5` |
| Checksum sidecar path fix | Merged in PR #74 |

## Why a new APK

Public downloads previously served `wallet-testnet-0.1.4` from a parallel branch that is
not an ancestor of the canonical integration line. This release publishes the wallet from
the merged go-live / canonical-integration `main`.

## Operator follow-up

Install the APK on a device and confirm:

1. Wallet connects to `https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com`
2. Faucet info/request works against the live bridge
3. Explorer links open the public website explorer

Test SUDH has no monetary value.
