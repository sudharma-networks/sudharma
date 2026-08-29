export interface NormalizedReleaseArtifact {
  id: string;
  slot: string;
  kind: string;
  name: string;
  version: string;
  channel: string;
  platform: string;
  architecture: string;
  status: "available";
  downloadUrl: string;
  sourceUrl: string;
  releaseNotesUrl?: string;
  checksumUrl?: string;
  sha256?: string;
  fileSize?: string;
  releaseDate?: string;
  prerelease?: boolean;
  safetyNote?: string;
}

export function classifyAsset(release: any, asset: any): NormalizedReleaseArtifact | null;
export function normalizeReleases(releases: any[]): NormalizedReleaseArtifact[];
export function projectStatus(releases: any[], commits: any[]): Record<string, string | null>;
export function sync(): Promise<void>;
