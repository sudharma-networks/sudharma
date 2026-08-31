import { DOWNLOADS, mergeDownloads, type DownloadArtifact } from "@/lib/downloads";

it("verified releases replace matching placeholders without duplication", () => {
  const generated = [{
    id: "wallet:test", slot: "android-wallet", kind: "wallet", name: "Wallet", version: "v1", channel: "testnet", platform: "Android", architecture: "arm64", status: "available", downloadUrl: "https://github.com/sudharma-networks/sudharma/releases/download/v1/wallet.apk", sourceUrl: "https://github.com/sudharma-networks/sudharma"
  }] as any;
  const fallback: DownloadArtifact[] = [
    { id: "android-wallet", kind: "wallet", name: "Wallet", version: "dev", channel: "testnet", platform: "Android", architecture: "arm64", status: "in-development" },
    { id: "sdk", kind: "developer", name: "SDK", version: "planned", channel: "development", platform: "Any", architecture: "Any", status: "planned" }
  ];
  const merged = mergeDownloads(generated, fallback);
  expect(merged.filter((item) => item.kind === "wallet")).toHaveLength(1);
  expect(merged.some((item) => item.id === "sdk")).toBe(true);
});

it("every public download has official provenance and unavailable items have no URL", () => {
  for (const artifact of DOWNLOADS) {
    if (artifact.status === "available") expect(artifact.sourceUrl ?? artifact.releaseNotesUrl).toContain("github.com/sudharma-networks/sudharma");
    else expect(artifact.downloadUrl).toBeUndefined();
  }
});

it("current public wallet and miners are synchronized", () => {
  expect(DOWNLOADS.some((item) => item.kind === "wallet" && item.status === "available" && item.version.startsWith("wallet-testnet-"))).toBe(true);
  expect(DOWNLOADS.filter((item) => item.kind === "miner" && item.status === "available")).toHaveLength(2);
  expect(DOWNLOADS.some((item) => item.id === "windows-gpu-miner" && item.status === "in-development")).toBe(true);
});
