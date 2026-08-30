# Same-Site Latest Wallet Download Design

Date: 2026-08-31
Status: Proposed, user-approved architecture awaiting implementation-plan review
Branch: `feature/website-foundation`

## Problem

The latest wallet release (`wallet-testnet-0.1.3`) is valid on GitHub, but Android users can be redirected from the GitHub release URL to `release-assets.githubusercontent.com`. On at least one real mobile connection this host times out, producing `ERR_CONNECTION_TIMED_OUT` even though the APK exists and is healthy.

The Sudharma website is built with Next.js static export (`output: "export"`), so it cannot provide a runtime server-side proxy route. A reliable fix therefore needs the wallet binary to be present in the website's own static output instead of redirecting users to GitHub's release-asset CDN.

## Goals

1. Make the latest Android testnet wallet downloadable from the same public website host that serves the Downloads page.
2. Preserve release provenance and SHA-256 verification data from the official GitHub release.
3. Automatically replace the website-facing wallet file when a newer official wallet release is published.
4. Keep only the latest wallet visible to normal website users.
5. Allow older GitHub wallet releases to be removed after the new path is proven live, while preserving their Git tags/history unless explicitly requested otherwise.
6. Keep the change limited to wallet distribution; do not alter miner, node, faucet, blockchain, or wallet runtime behavior.

## Non-goals

- No new application server.
- No S3/CloudFront migration in this change.
- No wallet signing or build-process changes.
- No automatic deletion of Git tags.
- No deletion of older releases before the new download is verified live.

## Architecture

### Source of truth

GitHub Releases remains the authoritative source for wallet version, release notes, asset digest, and publication metadata.

The existing release-sync script continues to classify the newest official Android wallet. For that wallet only, the sync job additionally downloads the APK and `.sha256` sidecar during CI and publishes them into fixed static website paths:

- `web/public/downloads/Sudharma-Wallet-latest.apk`
- `web/public/downloads/Sudharma-Wallet-latest.apk.sha256`

The website metadata for the Android wallet will use `/downloads/Sudharma-Wallet-latest.apk` as its user-facing `downloadUrl` and the same-site checksum path as `checksumUrl`. The release-notes URL remains the original GitHub Release page.

### Why a fixed filename

A fixed `latest` filename means the Downloads page never needs to know the final website host or deployment URL and future wallet releases do not require UI changes. Each sync replaces the previous file contents, so normal users see only one wallet download target.

The artifact metadata still shows the real release tag/version (for example `wallet-testnet-0.1.3`) and SHA-256 from GitHub, so the fixed filename does not hide which build is being distributed.

## Release-sync data flow

1. `release: published` triggers `.github/workflows/sync-website-releases.yml`.
2. Workflow checks out `feature/website-foundation`.
3. `sync-github-releases.mjs` fetches public GitHub release metadata and identifies the newest Android wallet asset.
4. The sync process downloads only that newest APK and its checksum sidecar using the authenticated GitHub workflow context.
5. Before publishing, the job verifies the downloaded APK SHA-256 against the GitHub release digest.
6. If verification fails, the workflow exits non-zero and does not commit a new website wallet binary or metadata.
7. On success, the APK and sidecar are written to the fixed `web/public/downloads/` paths.
8. `github-releases.json` points the Android wallet artifact to the same-site static paths while retaining GitHub release notes and provenance.
9. The workflow commits the changed metadata and wallet files to `feature/website-foundation`.
10. Existing website hosting publishes that branch through its current external deployment mechanism.

## Failure handling and safety

The sync must fail closed. A new wallet binary is never exposed through the website unless its downloaded bytes match the SHA-256 digest published by GitHub for that release asset.

If GitHub is temporarily unavailable during sync, the workflow fails and leaves the previously verified website wallet in place. This is preferable to replacing a known-good file with partial or unverified bytes.

The website continues to display the testnet safety warning and official GitHub release-notes/provenance link.

## Repository-size trade-off

The current APK is about 20 MB. Committing successive binary replacements increases Git repository history even though only one file is present at the branch tip. This is acceptable as a short-term reliability fix, but not ideal indefinitely.

A later migration can move the fixed `/downloads/Sudharma-Wallet-latest.apk` delivery target to S3/CloudFront or another object store without changing the user-facing Downloads UI concept. That migration is explicitly outside this change.

## Older wallet releases

After the same-site `0.1.3` APK is verified from the live website on a real mobile connection, older wallet GitHub Releases may be deleted so the Releases page shows only the latest wallet release.

Deletion scope:

- Delete older wallet **Release objects/assets** such as `wallet-testnet-0.1.2` and `wallet-testnet-v0.1.0`.
- Keep their Git tags by default, preserving source-history anchors and development provenance.
- Do not delete non-wallet releases.
- Do not delete `wallet-testnet-0.1.3`.

Because the connected GitHub API surface does not expose direct release deletion, cleanup will be performed only through an explicitly reviewed repository workflow/API step or by the repository owner in GitHub UI. No hidden or broad deletion mechanism will be introduced.

## Testing strategy

Implementation follows TDD.

Unit tests will first demonstrate that the latest Android wallet metadata uses the same-site `/downloads/Sudharma-Wallet-latest.apk` path while preserving release tag, digest, notes URL, and checksum path.

Sync tests will cover:

- only the newest Android wallet is selected;
- checksum sidecars are paired with the correct APK;
- a digest mismatch rejects publication;
- non-wallet release classification remains unchanged;
- older wallet releases do not produce multiple website wallet cards.

Website CI must pass typecheck, unit tests, production static build, and Playwright E2E tests before the change is considered code-complete.

## Live verification

The change is not considered operationally complete until all of the following are true:

1. the website branch contains the verified latest APK and checksum;
2. website CI is green;
3. the live Downloads page points to the same-site wallet path;
4. the same-site APK returns successfully from the public website host;
5. the APK SHA-256 fetched from the live path equals the GitHub release digest;
6. a real Android/mobile download succeeds without redirecting to `release-assets.githubusercontent.com`.

Only after these checks pass may old wallet GitHub Release objects be removed.
