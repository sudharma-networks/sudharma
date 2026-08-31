# Android wallet 0.1.5 go-live release

**Recorded:** 2026-08-31  
**Branch:** `cursor/android-wallet-0.1.5-8441`  
**Base:** `main` at `14d15af` (PR #69)

## Why a new APK

The live downloads page currently serves `wallet-testnet-0.1.4`, which was published
from a parallel branch (`cursor/wallet-txn-details-83d4`) that is **not** an ancestor
of the canonical integration line on `main`. Stage 5 step 7 deferred APK publish until
RPC/faucet/explorer were stable. Those surfaces are live, so this release publishes the
wallet from the merged go-live commit.

## Changes on this branch

- Bump `versionCode` to `5` and `versionName` to `0.1.5-testnet`
- Name CI artifacts from `versionName` instead of a hard-coded `0.1.0`
- Add manual `Android Wallet Publish` workflow (`confirm=PUBLISH` required)
- Keep automatic Android CI release-free (existing safety guard)

## Operator publish (after this PR is merged to main)

```powershell
gh workflow run android-wallet-publish.yml -R sudharma-networks/sudharma --ref main -f tag=wallet-testnet-0.1.5 -f confirm=PUBLISH
```

Then refresh website downloads:

```powershell
gh workflow run sync-website-releases.yml -R sudharma-networks/sudharma
```

## Verify

- Release page: `https://github.com/sudharma-networks/sudharma/releases/tag/wallet-testnet-0.1.5`
- Downloads page shows wallet version `wallet-testnet-0.1.5`
- Install APK, confirm public RPC/faucet reachability against
  `https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com`

## Notes

- This is a **debug** APK (`assembleDebug`), matching prior public testnet wallet releases.
- Test SUDH has no monetary value.
