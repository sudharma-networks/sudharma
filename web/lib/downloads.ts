import releaseSnapshot from "../public/data/github-releases.json";

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
  checksumUrl?: string;
  releaseDate?: string;
  downloadUrl?: string;
  releaseNotesUrl?: string;
  sourceUrl?: string;
  status: DownloadStatus;
  safetyNote?: string;
}

interface GeneratedDownloadArtifact extends DownloadArtifact {
  slot: "android-wallet" | "nvidia-miner" | "amd-miner" | "windows-gpu-miner" | "node-binary";
  releaseTag?: string;
  prerelease?: boolean;
}

const FALLBACK_DOWNLOADS: DownloadArtifact[] = [
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
  { id: "nvidia-miner", kind: "miner", name: "Sudharma NVIDIA Miner", version: "pre-release", channel: "experimental", platform: "Windows", architecture: "CUDA GPU", status: "in-development" },
  { id: "amd-miner", kind: "miner", name: "Sudharma AMD / OpenCL Miner", version: "pre-release", channel: "experimental", platform: "Windows", architecture: "OpenCL GPU", status: "in-development" },
  { id: "windows-gpu-miner", kind: "miner", name: "Sudharma One-Click Windows GPU Miner", version: "pre-release", channel: "experimental", platform: "Windows", architecture: "NVIDIA CUDA / AMD OpenCL GPU", status: "in-development" },
  { id: "node-binary", kind: "node", name: "Sudharma Node Binary", version: "pre-mainnet", channel: "development", platform: "Linux", architecture: "x86_64", status: "in-development" },
  { id: "sdk", kind: "developer", name: "Sudharma SDKs", version: "planned", channel: "development", platform: "Cross-platform", architecture: "Multiple", status: "planned" }
];

const slotForFallback = (artifact: DownloadArtifact) => {
  if (artifact.id === "android-wallet") return "android-wallet";
  if (artifact.id === "nvidia-miner") return "nvidia-miner";
  if (artifact.id === "amd-miner") return "amd-miner";
  if (artifact.id === "windows-gpu-miner") return "windows-gpu-miner";
  if (artifact.id === "node-binary") return "node-binary";
  return undefined;
};

export function mergeDownloads(generated: GeneratedDownloadArtifact[], fallback = FALLBACK_DOWNLOADS): DownloadArtifact[] {
  const official = generated.filter((artifact) => artifact.status === "available" && artifact.downloadUrl?.startsWith("https://github.com/sudharma-networks/sudharma/releases/download/"));
  const promoted = new Set(official.map((artifact) => artifact.slot));
  const remaining = fallback.filter((artifact) => {
    const slot = slotForFallback(artifact);
    return !slot || !promoted.has(slot);
  });
  return [...official, ...remaining];
}

export const DOWNLOADS: DownloadArtifact[] = mergeDownloads(releaseSnapshot.artifacts as GeneratedDownloadArtifact[]);

export function filterDownloads(kind: DownloadKind) {
  return DOWNLOADS.filter((artifact) => artifact.kind === kind);
}
