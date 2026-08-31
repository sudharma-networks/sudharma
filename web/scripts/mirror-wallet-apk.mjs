import { createHash } from "node:crypto";
import { mkdir, readFile, rename, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const REPO = "sudharma-networks/sudharma";
const APK_NAME = "Sudharma-Wallet-latest.apk";
const CHECKSUM_NAME = `${APK_NAME}.sha256`;

function repoRoot() {
  return path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
}

function officialAssetUrl(tag, filename) {
  return `https://github.com/${REPO}/releases/download/${encodeURIComponent(tag)}/${encodeURIComponent(filename)}`;
}

function apkFilenameFromArtifact(artifact) {
  const id = String(artifact?.id || "");
  const parts = id.split(":");
  return parts.length >= 3 ? parts.slice(2).join(":") : "";
}

export function walletMirrorPlan(snapshot) {
  const wallet = (snapshot?.artifacts || []).find((artifact) => artifact.slot === "android-wallet" && artifact.status === "available");
  if (!wallet?.releaseTag) return null;
  const filename = apkFilenameFromArtifact(wallet);
  if (!filename.endsWith(".apk")) return null;
  return {
    tag: wallet.releaseTag,
    apkUrl: officialAssetUrl(wallet.releaseTag, filename),
    checksumUrl: officialAssetUrl(wallet.releaseTag, `${filename}.sha256`),
    expectedSha256: typeof wallet.sha256 === "string" ? wallet.sha256.toLowerCase() : undefined
  };
}

async function download(url) {
  const headers = { "User-Agent": "sudharma-website-wallet-mirror", Accept: "application/octet-stream" };
  if (process.env.GITHUB_TOKEN) headers.Authorization = `Bearer ${process.env.GITHUB_TOKEN}`;
  const response = await fetch(url, { headers, redirect: "follow" });
  if (!response.ok) {
    throw new Error(`download ${url} failed: HTTP ${response.status}`);
  }
  return Buffer.from(await response.arrayBuffer());
}

function sha256(buffer) {
  return createHash("sha256").update(buffer).digest("hex");
}

async function writeAtomic(target, data) {
  await mkdir(path.dirname(target), { recursive: true });
  const temp = `${target}.tmp`;
  await writeFile(temp, data);
  await rename(temp, target);
}

export async function mirrorWalletApk({ root = repoRoot(), snapshotPath } = {}) {
  if (process.env.SKIP_WALLET_MIRROR === "1") {
    return { skipped: true };
  }

  const resolvedSnapshot = snapshotPath || path.join(root, "public/data/github-releases.json");
  const snapshot = JSON.parse(await readFile(resolvedSnapshot, "utf8"));
  const plan = walletMirrorPlan(snapshot);
  if (!plan) {
    throw new Error("no public Android wallet artifact found to mirror");
  }

  const apkDir = path.join(root, "public/downloads");
  const apkPath = path.join(apkDir, APK_NAME);
  const checksumPath = path.join(apkDir, CHECKSUM_NAME);

  try {
    const existing = await readFile(apkPath);
    if (plan.expectedSha256 && sha256(existing) === plan.expectedSha256) {
      return { skipped: true, reason: "already-mirrored", tag: plan.tag };
    }
  } catch {
    // Download a fresh copy.
  }

  const apk = await download(plan.apkUrl);
  const digest = sha256(apk);
  if (plan.expectedSha256 && digest !== plan.expectedSha256) {
    throw new Error(`mirrored APK sha256 ${digest} != ${plan.expectedSha256}`);
  }

  await writeAtomic(apkPath, apk);
  await writeAtomic(checksumPath, Buffer.from(`${digest}  ${APK_NAME}\n`, "utf8"));
  return { skipped: false, tag: plan.tag, sha256: digest, apkPath };
}

if (process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url) {
  mirrorWalletApk().then((result) => {
    console.log(JSON.stringify(result));
  }).catch((error) => {
    console.error(error);
    process.exitCode = 1;
  });
}
