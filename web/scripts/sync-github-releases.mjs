import { mkdir, readFile, rename, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const REPO = "sudharma-networks/sudharma";
const OFFICIAL = `https://github.com/${REPO}`;
const WALLET_PUBLIC_PATH = "/downloads/Sudharma-Wallet-latest.apk";
const WALLET_CHECKSUM_PATH = `${WALLET_PUBLIC_PATH}.sha256`;

function bytes(size) {
  if (!Number.isFinite(size)) return undefined;
  const mb = size / 1024 / 1024;
  return mb >= 1 ? `${mb.toFixed(1)} MB` : `${Math.max(1, Math.round(size / 1024))} KB`;
}

function sha256(digest) {
  return typeof digest === "string" && digest.startsWith("sha256:") ? digest.slice(7) : undefined;
}

export function classifyAsset(release, asset) {
  const filename = String(asset?.name || "");
  const lower = filename.toLowerCase();
  const context = `${release?.tag_name || ""} ${release?.name || ""}`.toLowerCase();
  if (!asset?.browser_download_url?.startsWith(`${OFFICIAL}/releases/download/`)) return null;
  if (lower.endsWith(".sha256") || lower.includes("sha256sums") || lower.includes("source-revision")) return null;

  let slot;
  let kind;
  let channel;
  let name;
  let platform;
  let architecture;
  let safetyNote;

  if (lower.endsWith(".apk") && context.includes("wallet")) {
    slot = "android-wallet"; kind = "wallet"; channel = "testnet";
    name = "Sudharma Android Wallet"; platform = "Android"; architecture = "arm64 / compatible";
    safetyNote = "Public testnet software. Do not use it for assets of real-world value.";
  } else if (lower.includes("gpu-miner") && lower.includes("windows")) {
    slot = "windows-gpu-miner"; kind = "miner"; channel = "experimental";
    name = "Sudharma One-Click Windows GPU Miner"; platform = "Windows"; architecture = "NVIDIA CUDA / AMD OpenCL GPU";
    safetyNote = "GPU-only miner. CPU and ASIC mining are not supported on public testnet or mainnet.";
  } else if ((lower.includes("nvidia") || lower.includes("cuda")) && lower.includes("miner")) {
    slot = "nvidia-miner"; kind = "miner"; channel = "experimental";
    name = "Khushi Miner — NVIDIA / CUDA"; platform = lower.includes("windows") ? "Windows" : "Cross-platform"; architecture = "NVIDIA CUDA GPU";
    safetyNote = "Experimental pre-mainnet miner. Unrestricted network mining remains gated.";
  } else if ((lower.includes("opencl") || lower.includes("amd")) && lower.includes("miner")) {
    slot = "amd-miner"; kind = "miner"; channel = "experimental";
    name = "Khushi Miner — AMD / OpenCL"; platform = lower.includes("windows") ? "Windows" : "Cross-platform"; architecture = "OpenCL GPU";
    safetyNote = "Experimental pre-mainnet miner. Unrestricted network mining remains gated.";
  } else {
    return null;
  }

  return {
    id: `${slot}:${release.tag_name}:${filename}`,
    slot,
    kind,
    name,
    version: release.tag_name,
    channel,
    platform,
    architecture,
    fileSize: bytes(Number(asset.size)),
    sha256: sha256(asset.digest),
    releaseDate: release.published_at || undefined,
    downloadUrl: asset.browser_download_url,
    releaseNotesUrl: release.html_url,
    sourceUrl: OFFICIAL,
    status: "available",
    releaseTag: release.tag_name,
    prerelease: Boolean(release.prerelease),
    safetyNote
  };
}

export function normalizeReleases(releases) {
  const output = [];
  const seenSlots = new Set();
  for (const release of [...releases].sort((a, b) => String(b.published_at || "").localeCompare(String(a.published_at || "")))) {
    for (const asset of release.assets || []) {
      const item = classifyAsset(release, asset);
      if (!item || seenSlots.has(item.slot)) continue;
      const sidecar = (release.assets || []).find((candidate) => candidate.name === `${asset.name}.sha256`);
      if (sidecar?.browser_download_url?.startsWith(`${OFFICIAL}/releases/download/`)) item.checksumUrl = sidecar.browser_download_url;
      output.push(item);
      seenSlots.add(item.slot);
    }
  }
  return output;
}

export function withSameSiteWalletUrls(artifacts) {
  return artifacts.map((artifact) => artifact.slot === "android-wallet"
    ? { ...artifact, downloadUrl: WALLET_PUBLIC_PATH, checksumUrl: WALLET_CHECKSUM_PATH }
    : artifact);
}

const RETRYABLE_STATUS = new Set([403, 429, 500, 502, 503, 504]);

export function isRetryableGitHubStatus(status) {
  return RETRYABLE_STATUS.has(status);
}

async function githubJson(endpoint, options = {}) {
  const { attempts = 4, delayMs = 2000, sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms)), fetchImpl = fetch } = options;
  const base = process.env.GITHUB_API_URL || "https://api.github.com";
  const headers = { Accept: "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28", "User-Agent": "sudharma-website-sync" };
  if (process.env.GITHUB_TOKEN) headers.Authorization = `Bearer ${process.env.GITHUB_TOKEN}`;

  let lastStatus = 0;
  for (let attempt = 1; attempt <= attempts; attempt += 1) {
    const response = await fetchImpl(`${base}${endpoint}`, { headers });
    if (response.ok) return response.json();
    lastStatus = response.status;
    if (!isRetryableGitHubStatus(response.status) || attempt === attempts) break;
    await sleep(delayMs * 2 ** (attempt - 1));
  }
  throw new Error(`GitHub API ${lastStatus}: ${endpoint}`);
}

export const __githubJsonForTests = githubJson;

async function atomicJson(target, value) {
  await mkdir(path.dirname(target), { recursive: true });
  const temp = `${target}.tmp`;
  await writeFile(temp, `${JSON.stringify(value, null, 2)}\n`, "utf8");
  JSON.parse(await readFile(temp, "utf8"));
  await rename(temp, target);
}

export function projectStatus(releases, commits) {
  const latestRelease = [...releases].filter((r) => r.published_at).sort((a, b) => b.published_at.localeCompare(a.published_at))[0];
  const latestCommit = commits?.[0];
  const releaseAt = latestRelease?.published_at || "";
  const commitAt = latestCommit?.commit?.committer?.date || latestCommit?.commit?.author?.date || "";
  return {
    source: "GitHub public repository metadata",
    generatedAt: [releaseAt, commitAt].filter(Boolean).sort().at(-1) || null,
    latestReleaseTag: latestRelease?.tag_name || null,
    latestReleaseUrl: latestRelease?.html_url || null,
    latestCommitSha: latestCommit?.sha || null,
    latestCommitUrl: latestCommit?.html_url || null,
    latestCommitAt: commitAt || null
  };
}

export async function sync() {
  const releases = await githubJson(`/repos/${REPO}/releases?per_page=100`);
  const commits = await githubJson(`/repos/${REPO}/commits?sha=main&per_page=1`);
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
  await atomicJson(path.join(root, "public/data/github-releases.json"), { schemaVersion: 1, repository: REPO, artifacts: normalizeReleases(releases) });
  await atomicJson(path.join(root, "public/data/project-status.json"), projectStatus(releases, commits));
}

if (process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url) {
  sync().catch((error) => { console.error(error); process.exitCode = 1; });
}
