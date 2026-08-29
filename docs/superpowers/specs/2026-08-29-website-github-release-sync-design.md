# Sudharma Website GitHub Release Auto-Sync Design

## Goal

Make the Sudharma public website automatically reflect public GitHub releases and downloadable artifacts without requiring manual edits to website source code after each wallet, miner, node or related public test release.

## Scope

The first implementation focuses on public release/download synchronization and a small public project-activity surface. It does not expose private repository data, secrets, workflow tokens, credentials, unpublished artifacts, or private security reports.

## Source of Truth

GitHub Releases for `sudharma-networks/sudharma` are the authoritative source for public downloadable binaries and release metadata.

The website may surface both normal releases and GitHub prereleases. Prereleases must never be presented as production-ready. Public prereleases are displayed with explicit lifecycle labels such as `TESTNET`, `EXPERIMENTAL`, or `PRE-MAINNET` according to release/tag/artifact classification.

## Architecture

Use a hybrid design:

1. A scheduled GitHub Actions workflow runs once per hour and fetches public GitHub release metadata for the repository.
2. The workflow normalizes releases and assets into a static JSON snapshot committed to the website branch under `web/public/data/github-releases.json` only when content changes.
3. The website reads the generated static snapshot during the normal Next.js static-export build. This keeps AWS Amplify hosting fully static and avoids exposing a GitHub token in browser code.
4. The website keeps a small hard-coded fallback manifest for planned/in-development items that do not yet have a public GitHub release asset.
5. Real public release assets override matching fallback entries and become downloadable automatically.
6. The sync workflow can also run on `workflow_dispatch` for immediate manual refresh and on relevant release events where supported, while the hourly schedule provides the periodic safety net requested by the project owner.

## Why Hourly Snapshot Instead of Browser GitHub API Calls

The current site is a Next.js static export hosted through AWS Amplify. A generated static snapshot preserves that architecture, avoids GitHub API rate-limit dependence in every visitor's browser, avoids CORS/runtime availability concerns, and ensures no GitHub token is shipped to clients.

## Release Classification

Public assets are classified conservatively.

- Android APK assets matching wallet release/tag conventions become `kind: wallet`, `channel: testnet` unless an explicitly stable production release convention is introduced later.
- NVIDIA/CUDA miner assets become `kind: miner`, `channel: experimental` while GPU mining remains pre-mainnet/test-stage.
- AMD/OpenCL miner assets become `kind: miner`, `channel: experimental` while GPU mining remains pre-mainnet/test-stage.
- Node binaries become `kind: node`, with `development` or `testnet` channel according to the release.
- Repository/source archives remain `kind: source`.
- Unknown binary assets must not automatically become prominent download cards. They remain visible only through their GitHub release page until an explicit classifier is added.

## Download Provenance

Every `available` download shown by the website must originate from a public GitHub Release asset URL or another explicitly verified official source.

Where GitHub provides an asset digest, the normalized snapshot stores it. SHA256 sidecar files are also linked when present. The website displays checksum/provenance metadata when available.

The website must never invent URLs, versions, file sizes, checksums or release dates.

## Safety and Status Rules

- Prerelease assets remain clearly labeled `TESTNET`, `EXPERIMENTAL`, or `PRE-MAINNET`.
- The Android wallet release must state that it is public testnet software and not for assets of real-world value.
- Wallet download pages continue warning users never to share seed phrases or private keys.
- Experimental mining packages must not imply unrestricted production mining if consensus/network activation is still gated.
- Unreleased planned items remain `in-development` or `planned` and have no active download URL.

## Data Model

The existing `DownloadArtifact` model remains the website-facing interface:

```ts
export type DownloadChannel = "stable" | "testnet" | "experimental" | "development";
export type DownloadKind = "wallet" | "miner" | "node" | "source" | "developer";
export type DownloadStatus = "available" | "in-development" | "planned";

export interface DownloadArtifact {
  id: string;
  kind: DownloadKind;
  name: string;
  version: string;
  channel: DownloadChannel;
  platform: string;
  architecture: string;
  fileSize?: string;
  sha256?: string;
  releaseDate?: string;
  downloadUrl?: string;
  releaseNotesUrl?: string;
  sourceUrl?: string;
  status: DownloadStatus;
}
```

The generated JSON snapshot uses an equivalent serializable shape plus optional GitHub release identifiers useful for deterministic merging.

## Merge Behavior

The website combines generated release artifacts with fallback planned items by stable artifact IDs/categories.

If a verified public GitHub asset exists for a product category, that real asset is rendered as `available`. The old placeholder for that product is removed from the rendered list rather than duplicated.

If no verified public asset exists, the fallback item remains visible with `in-development` or `planned` status and no download button.

Multiple public versions may be retained where useful, but the newest matching testnet/experimental artifact is displayed first.

## Project Activity Surface

The same scheduled workflow may generate a small `web/public/data/project-status.json` snapshot containing only safe public metadata such as the latest public release, latest public repository commit timestamp/SHA, and snapshot generation time. This is informational only and must not be used to claim features are complete or production-ready.

## Scheduling

The GitHub Actions sync workflow runs:

- hourly via `schedule` (`cron`),
- manually via `workflow_dispatch`,
- and may additionally respond to public release publication events if GitHub Actions event support fits the repository workflow.

Hourly is the primary periodic guarantee. The workflow must avoid creating a commit when generated content is unchanged.

## AWS Amplify Interaction

When the sync workflow commits an updated snapshot to `feature/website-foundation`, AWS Amplify's existing branch integration automatically rebuilds and deploys the website. No manual AWS action is required.

## Failure Handling

If GitHub metadata retrieval fails, the workflow exits without replacing the last known-good snapshot. The currently deployed website therefore continues serving the previous verified release data rather than an empty or corrupted download list.

If parsing/classification encounters an unknown asset, the workflow skips automatic promotion of that asset and preserves a trace in CI logs for future classifier updates.

## Testing

Tests must cover:

- public wallet APK release classification,
- NVIDIA/CUDA and OpenCL miner classification,
- prerelease lifecycle labels,
- checksum/digest propagation,
- unknown asset rejection from promoted download cards,
- planned-item fallback when no release exists,
- verified release overriding its placeholder without duplication,
- no `downloadUrl` on non-available artifacts,
- every available artifact having official provenance,
- static build using generated snapshots,
- desktop and mobile Downloads-page navigation/download controls.

The hourly workflow itself is validated with fixture-based parser tests plus a GitHub Actions dry generation step before committing output.

## Initial Verified Public Assets

The current repository already contains a public prerelease `wallet-testnet-v0.1.0` with Android APK `Sudharma-Wallet-0.1.0-testnet.apk`, a SHA256 sidecar, and release provenance. The auto-sync implementation should make that wallet automatically appear as an available public testnet download after the first successful snapshot generation.

The repository also contains the public `test-mining-v0.1.0` prerelease with NVIDIA and OpenCL Windows miner packages. These should appear as experimental/test-mining downloads with their published checksums/provenance and must retain the release's warning that unrestricted network mining remains gated.

## Non-Goals

This design does not automatically publish a build merely because a workflow artifact exists. A file becomes a public website download only after it is intentionally published as an official public GitHub Release asset (or another explicitly approved official release source).

It also does not merge the website feature branch into `main`, activate mainnet, enable unrestricted GPU mining, or expose backend/private project data.
