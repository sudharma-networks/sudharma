# Sudharma Website GitHub Release Auto-Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Automatically publish verified public GitHub release assets and safe public project activity to the static Sudharma website without manual website edits.

**Architecture:** A Node generator consumes GitHub Releases/repository metadata, classifies only known official assets, and writes deterministic static JSON snapshots under `web/public/data`. An hourly/release-triggered GitHub Actions workflow refreshes and commits changed snapshots to `feature/website-foundation`; the static Next.js site merges those verified artifacts with planned fallback entries, and Amplify redeploys the branch automatically.

**Tech Stack:** Node.js 22, TypeScript/JavaScript, Next.js 15 static export, Vitest, Playwright, GitHub Actions, GitHub REST API, AWS Amplify.

**Spec:** `docs/superpowers/specs/2026-08-29-website-github-release-sync-design.md`

## Global Constraints

- GitHub Releases for `sudharma-networks/sudharma` are the authoritative public binary source.
- Include public prereleases, but label them `TESTNET`, `EXPERIMENTAL`, or `PRE-MAINNET`; never present them as production-ready.
- Never expose GitHub tokens or credentials to browser code.
- Unknown binary assets are not promoted automatically.
- Non-available artifacts never have a `downloadUrl`.
- Every available artifact must have official provenance.
- Do not activate mainnet or unrestricted GPU mining.
- The workflow runs hourly, manually, and on public release publication; unchanged snapshots produce no commit.
- Failed retrieval/generation must preserve the last known-good snapshot.

---

### Task 1: Release snapshot generator and classifier

**Files:**
- Create: `web/scripts/sync-github-releases.mjs`
- Create: `web/tests/fixtures/github-releases.json`
- Create: `web/tests/github-release-sync.test.ts`
- Modify: `web/package.json`

**Interfaces:**
- Consumes: GitHub REST release JSON from `/repos/sudharma-networks/sudharma/releases`.
- Produces: `classifyAsset(release, asset)` and deterministic snapshot objects matching `DownloadArtifact` fields plus release provenance.

- [ ] **Step 1: Write failing fixture tests** covering wallet APK, SHA256 propagation, CUDA miner, OpenCL miner, prerelease labels, and rejection of an unknown `.bin` asset. Tests import the generator classifier and assert wallet => `kind: "wallet", channel: "testnet"`; CUDA/OpenCL => `kind: "miner", channel: "experimental"`; unknown => `null`.
- [ ] **Step 2: Run `npm --prefix web test -- github-release-sync.test.ts`** and verify RED because the generator does not exist.
- [ ] **Step 3: Implement `sync-github-releases.mjs`** with exported pure classification/normalization functions. Recognize `.apk` wallet assets only when release/tag/name contains wallet; recognize NVIDIA/CUDA and AMD/OpenCL miner archives by explicit naming; pair `.sha256` sidecars by basename; preserve GitHub `browser_download_url`, release `html_url`, published date, size and digest; skip unknown assets.
- [ ] **Step 4: Add `sync:github` to `web/package.json`** as `node scripts/sync-github-releases.mjs` and make CLI mode fetch `GITHUB_API_URL || https://api.github.com`, using `GITHUB_TOKEN` only server-side when present.
- [ ] **Step 5: Run the focused test and full `npm --prefix web test`**; both must pass.
- [ ] **Step 6: Commit** with `feat(web): generate verified GitHub release snapshot`.

### Task 2: Deterministic snapshots and safe fallback merge

**Files:**
- Create: `web/public/data/github-releases.json`
- Create: `web/public/data/project-status.json`
- Modify: `web/lib/downloads.ts`
- Create: `web/tests/download-merge.test.ts`

**Interfaces:**
- Consumes: generated `github-releases.json` and existing fallback artifacts.
- Produces: `getDownloads(): DownloadArtifact[]` where verified public assets replace matching placeholders without duplication.

- [ ] **Step 1: Write failing merge tests** asserting a verified Android wallet removes the `android-wallet` placeholder, verified NVIDIA/OpenCL releases remove their matching placeholders, planned SDK remains, non-available entries have no URL, and every available entry points to `github.com/sudharma-networks/sudharma` or another explicitly allowlisted official source.
- [ ] **Step 2: Run `npm --prefix web test -- download-merge.test.ts`** and verify RED against the current static array.
- [ ] **Step 3: Refactor `web/lib/downloads.ts`** into fallback constants plus a deterministic merge function keyed by product category; sort real releases newest first; retain source-code download; never infer missing metadata.
- [ ] **Step 4: Generate initial snapshots from current public GitHub data** so `wallet-testnet-v0.1.0` and the verified test-mining release appear when recognized.
- [ ] **Step 5: Run focused and full unit tests** and verify GREEN.
- [ ] **Step 6: Commit** with `feat(web): merge verified releases into downloads`.

### Task 3: Downloads UI provenance and lifecycle labels

**Files:**
- Modify: `web/components/download-card.tsx`
- Modify: `web/app/downloads/page.tsx`
- Modify/Create: `web/tests/downloads-page.test.tsx`

**Interfaces:**
- Consumes: merged `DownloadArtifact[]`.
- Produces: visible version, channel/status, size/date when known, checksum when known, official release notes/source links and active download only for verified `available` entries.

- [ ] **Step 1: Write failing UI tests** for `TESTNET` wallet labeling, `EXPERIMENTAL` miner labeling, checksum display, public testnet safety copy, and absence of active download buttons for planned/in-development entries.
- [ ] **Step 2: Run the focused test** and verify RED for missing release metadata behavior.
- [ ] **Step 3: Update the card/page** to render the verified metadata without inventing absent fields; keep the existing seed/private-key warning and add pre-mainnet/testnet warning near prerelease downloads.
- [ ] **Step 4: Run focused and full unit tests** and verify GREEN.
- [ ] **Step 5: Commit** with `feat(web): surface verified release provenance`.

### Task 4: Hourly and release-triggered synchronization workflow

**Files:**
- Create: `.github/workflows/sync-website-releases.yml`

**Interfaces:**
- Consumes: GitHub public release/repository API using workflow `GITHUB_TOKEN` with least privileges.
- Produces: updated snapshot commit on `feature/website-foundation` only when generated files differ.

- [ ] **Step 1: Create workflow triggers** for `schedule: cron: '17 * * * *'`, `workflow_dispatch`, and `release: types: [published, released, prereleased]`; set `contents: write` and no broader permissions.
- [ ] **Step 2: Checkout `feature/website-foundation`, setup Node 22, install web dependencies, run release-sync unit tests, run `npm --prefix web run sync:github`, and validate both JSON files with Node JSON parsing.**
- [ ] **Step 3: Add failure-safe commit logic** using `git diff --quiet -- web/public/data`; if unchanged exit successfully; if changed commit only the two snapshot files with message `chore(web): sync public GitHub releases` and push to `feature/website-foundation`.
- [ ] **Step 4: Ensure generation writes temp files and atomically renames only after successful fetch/parse**, so failures cannot replace last known-good data.
- [ ] **Step 5: Commit** with `ci(web): sync public releases hourly`.

### Task 5: Public project status snapshot

**Files:**
- Modify: `web/scripts/sync-github-releases.mjs`
- Create/Modify: `web/components/project-activity.tsx`
- Modify: `web/app/page.tsx`
- Create: `web/tests/project-activity.test.tsx`

**Interfaces:**
- Consumes: latest safe public release metadata and latest public repository commit metadata.
- Produces: informational project status showing snapshot time, latest public release, and abbreviated latest commit without claiming readiness.

- [ ] **Step 1: Write failing component tests** asserting the activity surface says public repository activity, displays abbreviated SHA/release tag, and never renders `production ready` from metadata.
- [ ] **Step 2: Extend generator** to write only safe fields: `generatedAt`, `latestReleaseTag`, `latestReleaseUrl`, `latestCommitSha`, `latestCommitUrl`, `latestCommitAt`.
- [ ] **Step 3: Implement project activity component** and place it on the homepage with explicit informational/pre-mainnet wording.
- [ ] **Step 4: Run focused/full unit tests** and verify GREEN.
- [ ] **Step 5: Commit** with `feat(web): show synchronized public project activity`.

### Task 6: Static build, mobile/desktop E2E, and live deployment verification

**Files:**
- Modify: `web/tests/e2e/navigation.spec.ts` or existing Playwright navigation spec
- Modify if needed: `.github/workflows/website-ci.yml`

**Interfaces:**
- Consumes: completed synchronized website.
- Produces: CI evidence that the static export and public download controls work on desktop and Pixel 7 viewport.

- [ ] **Step 1: Add E2E assertions** that `/downloads` contains the Android testnet wallet download, its link targets the official GitHub Release APK, experimental miner downloads retain warnings, and mobile navigation reaches Downloads.
- [ ] **Step 2: Run `npm --prefix web run typecheck`, `npm --prefix web test`, and `npm --prefix web run build`.** All must exit 0.
- [ ] **Step 3: Run Playwright desktop/mobile E2E** with the existing project configuration; all tests must pass.
- [ ] **Step 4: Push final branch state and inspect GitHub Actions** for Website CI and release-sync workflow success; do not claim success before completed green runs.
- [ ] **Step 5: Verify the AWS Amplify preview after its automatic rebuild** and confirm the public Downloads page exposes only the verified official assets.
- [ ] **Step 6: Commit any final test-only adjustment** with `test(web): verify synchronized downloads`.

## Self-Review

- Spec coverage: classifier, prereleases, provenance, checksums, unknown rejection, fallback merge, hourly/manual/release triggers, project status, failure preservation, Amplify/static build and mobile/desktop verification are all assigned.
- Placeholder scan: no implementation placeholders remain; each task has explicit behavior and commands.
- Type consistency: website-facing artifacts remain `DownloadArtifact`; generator output supplies the same fields and merge returns `DownloadArtifact[]`.
