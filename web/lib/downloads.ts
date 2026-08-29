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

export const DOWNLOADS: DownloadArtifact[] = [
  {
    id: "source-main",
    kind: "source",
    name: "Sudharma source code",
    version: "main",
    channel: "development",
    platform: "Source",
    architecture: "Any",
    downloadUrl: "https://github.com/sudharma-networks/sudharma/archive/refs/heads/main.zip",
    sourceUrl: "https://github.com/sudharma-networks/sudharma",
    status: "available"
  },
  { id: "android-wallet", kind: "wallet", name: "Sudharma Android Wallet", version: "pre-release", channel: "testnet", platform: "Android", architecture: "arm64 / compatible", status: "in-development" },
  { id: "nvidia-miner", kind: "miner", name: "Sudharma NVIDIA Miner", version: "pre-release", channel: "experimental", platform: "Windows / Linux", architecture: "CUDA GPU", status: "in-development" },
  { id: "amd-miner", kind: "miner", name: "Sudharma AMD / OpenCL Miner", version: "pre-release", channel: "experimental", platform: "Windows / Linux", architecture: "OpenCL GPU", status: "in-development" },
  { id: "node-binary", kind: "node", name: "Sudharma Node Binary", version: "pre-mainnet", channel: "development", platform: "Linux", architecture: "x86_64", status: "in-development" },
  { id: "sdk", kind: "developer", name: "Sudharma SDKs", version: "planned", channel: "development", platform: "Cross-platform", architecture: "Multiple", status: "planned" }
];

export function filterDownloads(kind: DownloadKind) {
  return DOWNLOADS.filter((artifact) => artifact.kind === kind);
}
