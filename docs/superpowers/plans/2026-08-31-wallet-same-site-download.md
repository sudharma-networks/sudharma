# Same-Site Latest Wallet Download Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish the newest verified Android testnet wallet at a fixed same-site static URL so mobile users no longer depend on GitHub's `release-assets.githubusercontent.com` delivery host.

**Architecture:** GitHub Releases remains the source of truth. The existing Node release-sync script will identify the newest wallet, download that APK during CI, verify its SHA-256 against GitHub's release digest, atomically write it to `web/public/downloads/Sudharma-Wallet-latest.apk`, and rewrite only the wallet artifact's public download/checksum URLs to same-site paths. The release-sync workflow will commit the verified binary, checksum, and metadata to `feature/website-foundation`; static hosting then serves the binary directly.

**Tech Stack:** Node.js 22, Next.js 15 static export, Vitest 3, Playwright 1.55, GitHub Actions, GitHub Releases API, Node `crypto` and `fs/promises`.

**Spec:** `docs/superpowers/specs/2026-08-31-wallet-same-site-download-design.md`

## Global Constraints

- GitHub Releases for `sudharma-networks/sudharma` remain authoritative for wallet version, release notes, digest, and publication metadata.
- Publish only the newest official Android wallet to the same-site static path.
- Fixed paths are exactly `/downloads/Sudharma-Wallet-latest.apk` and `/downloads/Sudharma-Wallet-latest.apk.sha256`.
- Never publish or commit a newly downloaded APK unless its computed SHA-256 equals the GitHub release digest.
- If retrieval or verification fails, leave the previous verified website wallet and metadata unchanged.
- Preserve the real release tag and GitHub release-notes URL in metadata.
- Do not change miner, node, faucet, blockchain, wallet runtime, signing, or mainnet behavior.
- Do not delete old wallet releases until the new same-site path is verified live.
- Keep old Git tags when old Release objects are later removed.

---

### Task 1: Same-site wallet metadata contract

**Files:**
- Modify: `web/tests/github-release-sync.test.ts`
- Modify: `web/scripts/sync-github-releases.mjs`

**Interfaces:**
- Consumes: normalized GitHub release objects and classified artifacts.
- Produces: `withSameSiteWalletUrls(artifacts)` returning a new artifacts array where only `slot === "android-wallet"` receives fixed same-site `downloadUrl` and `checksumUrl` values.

- [ ] **Step 1: Write the failing metadata test** in `web/tests/github-release-sync.test.ts`:

```ts
import { classifyAsset, normalizeReleases, withSameSiteWalletUrls } from "../scripts/sync-github-releases.mjs";

it("publishes the newest Android wallet through fixed same-site URLs", () => {
  const [wallet] = withSameSiteWalletUrls(normalizeReleases([release]));
  expect(wallet.version).toBe("wallet-testnet-v0.1.0");
  expect(wallet.sha256).toBe("f4d0ec7898bcfd19a857a9930f71a2433c297112e4b1589b6856c1d397d8ebab");
  expect(wallet.downloadUrl).toBe("/downloads/Sudharma-Wallet-latest.apk");
  expect(wallet.checksumUrl).toBe("/downloads/Sudharma-Wallet-latest.apk.sha256");
  expect(wallet.releaseNotesUrl).toBe(release.html_url);
});
```

Also assert a CUDA miner retains its original GitHub release URL after `withSameSiteWalletUrls`.

- [ ] **Step 2: Run RED verification**:

```bash
npm --prefix web test -- github-release-sync.test.ts
```

Expected: FAIL because `withSameSiteWalletUrls` is not exported.

- [ ] **Step 3: Implement the minimal pure transform** in `web/scripts/sync-github-releases.mjs`:

```js
const WALLET_PUBLIC_PATH = "/downloads/Sudharma-Wallet-latest.apk";
const WALLET_CHECKSUM_PATH = `${WALLET_PUBLIC_PATH}.sha256`;

export function withSameSiteWalletUrls(artifacts) {
  return artifacts.map((artifact) => artifact.slot === "android-wallet"
    ? { ...artifact, downloadUrl: WALLET_PUBLIC_PATH, checksumUrl: WALLET_CHECKSUM_PATH }
    : artifact);
}
```

Do not change `classifyAsset` behavior.

- [ ] **Step 4: Run GREEN verification**:

```bash
npm --prefix web test -- github-release-sync.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit**:

```bash
git add web/tests/github-release-sync.test.ts web/scripts/sync-github-releases.mjs
git commit -m "test(web): define same-site wallet metadata"
```

### Task 2: Verified wallet mirroring

**Files:**
- Modify: `web/tests/github-release-sync.test.ts`
- Modify: `web/scripts/sync-github-releases.mjs`

**Interfaces:**
- Consumes: raw release list and an injected `fetchImpl` for testability.
- Produces: `findLatestWalletReleaseAsset(releases)` returning `{ release, asset, sidecar } | null`; `verifySha256(bytes, expectedHex)` returning boolean; `mirrorLatestWallet(releases, options)` that writes only verified bytes to fixed paths.

- [ ] **Step 1: Add failing selection and digest tests**:

```ts
import { findLatestWalletReleaseAsset, verifySha256 } from "../scripts/sync-github-releases.mjs";
import { createHash } from "node:crypto";

it("selects only the newest Android wallet asset and its matching sidecar", () => {
  const older = { ...release, published_at: "2026-08-28T00:00:00Z" };
  const newer = { ...release, tag_name: "wallet-testnet-0.1.3", published_at: "2026-08-30T19:06:50Z" };
  const selected = findLatestWalletReleaseAsset([older, newer]);
  expect(selected?.release.tag_name).toBe("wallet-testnet-0.1.3");
  expect(selected?.asset.name.endsWith(".apk")).toBe(true);
  expect(selected?.sidecar.name).toBe(`${selected?.asset.name}.sha256`);
});

it("rejects bytes whose SHA256 does not match the release digest", () => {
  const bytes = new TextEncoder().encode("wallet-bytes");
  const digest = createHash("sha256").update(bytes).digest("hex");
  expect(verifySha256(bytes, digest)).toBe(true);
  expect(verifySha256(bytes, "0".repeat(64))).toBe(false);
});
```

- [ ] **Step 2: Run RED verification**:

```bash
npm --prefix web test -- github-release-sync.test.ts
```

Expected: FAIL for missing exports.

- [ ] **Step 3: Implement selection and digest helpers**. Selection must reuse `classifyAsset(release, asset)` and sort by `published_at` descending. `verifySha256` must compute `createHash("sha256").update(bytes).digest("hex")` and compare lowercase hex strings exactly.

- [ ] **Step 4: Add the failing mirror test** using `mkdtemp`, a fake `fetchImpl`, and temporary output paths. The fake APK response returns known bytes; the fake sidecar response returns text. Assert the verified bytes and sidecar are written only when the digest matches. Add a second case where the digest is wrong and assert both target files remain absent.

- [ ] **Step 5: Run RED verification** and confirm the mirror test fails because `mirrorLatestWallet` is missing.

- [ ] **Step 6: Implement `mirrorLatestWallet` fail-closed**:
  - download APK and sidecar into memory;
  - require a valid 64-character release digest;
  - verify APK bytes before any target replacement;
  - create `public/downloads` as needed;
  - write temporary files in the target directory;
  - rename temp APK and temp sidecar to final fixed names only after both downloads and digest verification succeed;
  - delete temp files on failure and rethrow;
  - return the selected release tag/digest for logging.

The production call must pass authentication headers derived from `GITHUB_TOKEN` only inside Node/CI; no token enters generated JSON.

- [ ] **Step 7: Run focused GREEN verification**:

```bash
npm --prefix web test -- github-release-sync.test.ts
```

Expected: all sync tests PASS.

- [ ] **Step 8: Commit**:

```bash
git add web/tests/github-release-sync.test.ts web/scripts/sync-github-releases.mjs
git commit -m "feat(web): mirror verified latest wallet"
```

### Task 3: Wire verified mirroring into sync output

**Files:**
- Modify: `web/scripts/sync-github-releases.mjs`
- Modify: `web/tests/download-merge.test.ts`
- Modify: `web/lib/downloads.ts`
- Modify: `web/e2e/navigation.spec.ts`

**Interfaces:**
- Consumes: verified static wallet files and transformed release snapshot.
- Produces: one available wallet card using `/downloads/Sudharma-Wallet-latest.apk`; miners retain GitHub URLs and official provenance.

- [ ] **Step 1: Write failing merge expectations** so generated same-site wallet URLs are accepted as official when `sourceUrl`/`releaseNotesUrl` points to `github.com/sudharma-networks/sudharma`:

```ts
const generated = [{
  id: "wallet:test",
  slot: "android-wallet",
  kind: "wallet",
  name: "Wallet",
  version: "wallet-testnet-0.1.3",
  channel: "testnet",
  platform: "Android",
  architecture: "arm64",
  status: "available",
  downloadUrl: "/downloads/Sudharma-Wallet-latest.apk",
  checksumUrl: "/downloads/Sudharma-Wallet-latest.apk.sha256",
  releaseNotesUrl: "https://github.com/sudharma-networks/sudharma/releases/tag/wallet-testnet-0.1.3",
  sourceUrl: "https://github.com/sudharma-networks/sudharma"
}] as any;
```

Assert it replaces the wallet placeholder and remains available.

- [ ] **Step 2: Run RED verification**:

```bash
npm --prefix web test -- download-merge.test.ts
```

Expected: FAIL because `mergeDownloads` currently accepts only GitHub release-download URLs.

- [ ] **Step 3: Update `mergeDownloads` minimally** to allow an available `android-wallet` when its URL is exactly `/downloads/Sudharma-Wallet-latest.apk` and it retains official GitHub provenance. Keep the existing GitHub release-download allowlist for miners and other generated artifacts.

- [ ] **Step 4: Update `sync()`** in `sync-github-releases.mjs` in this order:
  1. fetch releases/commits;
  2. call `mirrorLatestWallet(releases, ...)` and fail if it cannot verify the newest wallet;
  3. call `normalizeReleases(releases)`;
  4. call `withSameSiteWalletUrls(...)`;
  5. atomically write JSON snapshots.

This ordering ensures JSON cannot advertise a new same-site wallet before verified bytes exist.

- [ ] **Step 5: Update Playwright expectation** in `web/e2e/navigation.spec.ts`:

```ts
expect(walletArtifact?.downloadUrl).toBe("/downloads/Sudharma-Wallet-latest.apk");
```

Continue asserting the Download button uses `walletArtifact.downloadUrl` and the wallet remains labeled `TESTNET`.

- [ ] **Step 6: Run unit tests**:

```bash
npm --prefix web test -- github-release-sync.test.ts download-merge.test.ts
npm --prefix web test
```

Expected: PASS.

- [ ] **Step 7: Commit**:

```bash
git add web/scripts/sync-github-releases.mjs web/tests/download-merge.test.ts web/lib/downloads.ts web/e2e/navigation.spec.ts
git commit -m "feat(web): serve wallet from same-site path"
```

### Task 4: Publish mirrored files through GitHub Actions

**Files:**
- Modify: `.github/workflows/sync-website-releases.yml`

**Interfaces:**
- Consumes: `npm --prefix web run sync:github` output.
- Produces: committed `web/public/downloads/Sudharma-Wallet-latest.apk`, `.sha256`, and updated JSON snapshots on `feature/website-foundation`.

- [ ] **Step 1: Add a safe push trigger** so the implementation commit can bootstrap the existing `0.1.3` release without waiting for another release:

```yaml
on:
  push:
    branches: [feature/website-foundation]
    paths:
      - '.github/workflows/sync-website-releases.yml'
      - 'web/scripts/sync-github-releases.mjs'
      - 'web/tests/github-release-sync.test.ts'
      - 'web/lib/downloads.ts'
  release:
    types: [published]
  workflow_dispatch:
```

Do not include `web/public/**` in the push paths; this prevents the bot's generated-data commit from recursively retriggering the sync.

- [ ] **Step 2: Add focused verification before syncing**:

```yaml
- run: npm install
- run: npm test -- github-release-sync.test.ts download-merge.test.ts
- run: npm run sync:github
```

- [ ] **Step 3: Stage the generated wallet files**:

```bash
git add public/data/github-releases.json public/data/project-status.json \
  public/downloads/Sudharma-Wallet-latest.apk \
  public/downloads/Sudharma-Wallet-latest.apk.sha256
```

Keep the existing no-diff success exit and push destination `feature/website-foundation`.

- [ ] **Step 4: Commit the workflow change**:

```bash
git add .github/workflows/sync-website-releases.yml
git commit -m "ci(web): publish verified wallet mirror"
```

The push itself should trigger the workflow once.

### Task 5: CI and live verification gate

**Files:**
- No production files unless verification reveals a defect.

**Interfaces:**
- Consumes: branch tip after the release-sync bot commit.
- Produces: evidence for code correctness and operational availability.

- [ ] **Step 1: Verify the release-sync workflow completed successfully** and its bot commit contains all four generated paths: two JSON files plus APK and checksum.

- [ ] **Step 2: Verify Website CI for that bot commit** completes with success for typecheck, unit tests, static build, Chromium installation, and Playwright desktop/mobile E2E.

- [ ] **Step 3: Verify branch metadata** shows wallet `version: "wallet-testnet-0.1.3"`, SHA-256 `51def7fe2d651d70289cc084e9593a516dc7a129ea32fb3a4b658b37f2235d20`, `downloadUrl: "/downloads/Sudharma-Wallet-latest.apk"`, and same-site checksum URL.

- [ ] **Step 4: Verify the live public website**:
  - `/downloads` renders the `0.1.3` wallet;
  - the Download link is same-origin;
  - requesting `/downloads/Sudharma-Wallet-latest.apk` returns successfully without redirecting to `release-assets.githubusercontent.com`;
  - live response size is approximately 20.9 MB;
  - SHA-256 of live bytes equals `51def7fe2d651d70289cc084e9593a516dc7a129ea32fb3a4b658b37f2235d20`.

- [ ] **Step 5: Ask for one real Android download check** only after automated live verification passes. Do not call the issue operationally complete until that mobile check succeeds.

### Task 6: Remove older wallet Release objects only after live success

**Files:**
- No tag deletion.
- Optional temporary reviewed workflow only if direct GitHub release deletion remains unavailable to the connected tool surface.

**Interfaces:**
- Consumes: verified live same-site `0.1.3` download and current GitHub Release list.
- Produces: GitHub Releases page retaining `wallet-testnet-0.1.3` while older wallet Release objects/assets are removed; Git tags remain intact.

- [ ] **Step 1: Re-fetch the complete release list immediately before deletion** and identify wallet releases older than `wallet-testnet-0.1.3`. Expected known candidates include `wallet-testnet-0.1.2` and `wallet-testnet-v0.1.0`; do not rely only on these names if additional older wallet releases exist.

- [ ] **Step 2: Assert deletion scope** before any mutation: keep release `wallet-testnet-0.1.3`; keep all non-wallet releases; keep all Git tags.

- [ ] **Step 3: Use the narrowest available deletion mechanism**. If the connector exposes direct release deletion at execution time, delete only the identified Release objects. Otherwise, create an explicitly reviewed one-time repository workflow that uses `gh api --method DELETE /repos/${GITHUB_REPOSITORY}/releases/<exact-id>` for the exact reviewed release IDs and does not call any tag endpoint.

- [ ] **Step 4: Verify after deletion** that the latest wallet release still has both `Sudharma-Wallet-0.1.3.apk` and its checksum, non-wallet releases remain, and old wallet Git tags still resolve.

- [ ] **Step 5: Remove any temporary one-time cleanup workflow** after successful cleanup and verify its deletion does not alter the mirrored APK or website metadata.

## Self-Review

- **Spec coverage:** same-site fixed paths, latest-only selection, GitHub provenance, digest verification, fail-closed atomic publication, static-export compatibility, release-triggered replacement, CI/E2E, live byte/digest verification, and delayed old-release cleanup are all covered.
- **Placeholder scan:** no `TBD`, `TODO`, or unspecified implementation step remains. Cleanup explicitly depends on live verification and exact release IDs gathered at execution time because deleting unreviewed IDs would violate the spec.
- **Type/interface consistency:** `classifyAsset` and `normalizeReleases` remain the classifier API; `withSameSiteWalletUrls`, `findLatestWalletReleaseAsset`, `verifySha256`, and `mirrorLatestWallet` are introduced once and referenced consistently. `DownloadArtifact.downloadUrl` remains a string and now accepts one explicit same-site wallet path in addition to existing official GitHub URLs.
