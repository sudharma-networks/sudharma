# Sudharma Website Design Amendment — Downloads Hub

Date: 2026-08-29
Status: Approved design amendment, pre-implementation
Repository: `sudharma-networks/sudharma`
Parent spec: `docs/superpowers/specs/2026-08-29-sudharma-website-design.md`

## Purpose

Add a first-class **Downloads** destination to the Sudharma website so users, miners, developers, and contributors can find verified Sudharma software and source resources in one trusted place.

## Navigation

Add `Downloads` to the primary navigation:

`Home | Network | SUDH | Wallet | Mining | Developers | Downloads | Testnet | Explorer | Faucet | Roadmap | Docs | Community | Support`

Primary route:

- `/downloads`

Optional filtered routes or query-state views may include:

- `/downloads?type=wallet`
- `/downloads?type=miner`
- `/downloads?type=node`
- `/downloads?type=source`
- `/downloads?type=developer`

The Downloads page must be reachable from the global header, Wallet Hub, Mining Hub, Developer Hub, and relevant documentation pages.

## Download Categories

The page groups verified artifacts into clear cards or sections:

### Wallets

- Android wallet APK when a verified release exists
- desktop/CLI wallet binaries when verified releases exist
- source-build link where appropriate
- version, release date, supported platform, architecture, readiness label, and checksum/signature information where available

### Miners

- NVIDIA/CUDA miner releases when verified
- AMD/OpenCL or cross-vendor miner releases when verified
- other GPU builds only after compatibility is verified
- version, supported operating system, supported architecture/GPU class, readiness label, release notes, and checksum/signature information where available

### Node Software

- Sudharma node binaries when verified public releases exist
- platform/architecture variants where available
- source-build instructions
- release notes

### Source Code

- official Sudharma GitHub repository
- tagged source releases or source archives when available
- build-from-source documentation
- developer/contribution links

### Developer Resources

- API/RPC examples when published
- SDKs only after they actually exist
- sample integrations only after they are versioned and verified
- protocol/developer documentation links

## Trust and Safety Requirements

The website must never link to unverified third-party binaries as official Sudharma downloads.

Each downloadable binary should display, when available:

- artifact name
- version
- release channel (`Stable`, `Testnet`, `Experimental`, or `Development`)
- operating system
- architecture
- file size
- release date
- SHA-256 checksum
- signature/verification information if implemented
- release notes
- source/release provenance

A prominent safety notice must advise users to download Sudharma software only from official Sudharma website/GitHub release sources and to verify checksums where supplied.

Development/testnet artifacts must not be presented as production-mainnet software.

## Empty and Unreleased States

If a planned artifact is not yet available, the page shows its status rather than a fake or dead download button. Example:

`AMD Miner — In Development`

`Download` controls appear only when a verified artifact URL exists.

## License and Open Development

The repository now contains the **Apache License, Version 2.0**. Website copy and source-download messaging should be updated to reflect the published repository license rather than the parent spec's earlier pre-license wording.

The site may explain that Sudharma source is available under Apache-2.0 terms, with a direct link to the repository `LICENSE` file. It must not summarize the license in a way that overrides or contradicts the actual license text.

## Data Source

Download metadata should come from a controlled versioned manifest or trusted GitHub Release metadata rather than being duplicated as arbitrary page text.

Recommended initial interface:

```ts
export type DownloadChannel = "stable" | "testnet" | "experimental" | "development";

export type DownloadKind = "wallet" | "miner" | "node" | "source" | "developer";

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
  status: "available" | "in-development" | "planned";
}
```

The UI renders only validated entries from this source.

## Support Integration

Every artifact card has a contextual `Report Download Problem` action. The report form receives safe metadata such as artifact ID, version, platform, and current page route.

Security issues related to a binary or release follow the private security-report route defined in the parent specification.

## Acceptance Criteria

- `Downloads` appears in global navigation.
- `/downloads` is a dedicated premium page, not a file dump.
- Wallet, Miner, Node, Source, and Developer categories are visibly separated.
- Only verified artifacts expose active download buttons.
- Development/unavailable artifacts show status labels instead of broken links.
- Available binary entries can display SHA-256 verification data.
- Source code points to the official repository and Apache-2.0 license.
- Relevant Wallet, Mining, and Developer pages deep-link into Downloads.
- Every artifact can be reported through contextual support.
- Mobile and desktop layouts remain fully usable.
