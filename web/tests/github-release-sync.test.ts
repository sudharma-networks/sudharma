import { classifyAsset, normalizeReleases } from "../scripts/sync-github-releases.mjs";

const release = {
  tag_name: "wallet-testnet-v0.1.0",
  name: "Sudharma Wallet 0.1.0 — Public Testnet",
  html_url: "https://github.com/sudharma-networks/sudharma/releases/tag/wallet-testnet-v0.1.0",
  prerelease: true,
  published_at: "2026-08-29T15:45:59Z",
  assets: [
    {
      name: "Sudharma-Wallet-0.1.0-testnet.apk",
      size: 20926171,
      digest: "sha256:f4d0ec7898bcfd19a857a9930f71a2433c297112e4b1589b6856c1d397d8ebab",
      browser_download_url: "https://github.com/sudharma-networks/sudharma/releases/download/wallet-testnet-v0.1.0/Sudharma-Wallet-0.1.0-testnet.apk"
    },
    {
      name: "Sudharma-Wallet-0.1.0-testnet.apk.sha256",
      size: 105,
      browser_download_url: "https://github.com/sudharma-networks/sudharma/releases/download/wallet-testnet-v0.1.0/Sudharma-Wallet-0.1.0-testnet.apk.sha256"
    }
  ]
};

it("classifies a public testnet Android wallet APK", () => {
  const item = classifyAsset(release, release.assets[0]);
  expect(item?.slot).toBe("android-wallet");
  expect(item?.kind).toBe("wallet");
  expect(item?.channel).toBe("testnet");
  expect(item?.status).toBe("available");
});

it("propagates the GitHub SHA256 digest and sidecar URL", () => {
  const [item] = normalizeReleases([release]);
  expect(item.sha256).toBe("f4d0ec7898bcfd19a857a9930f71a2433c297112e4b1589b6856c1d397d8ebab");
  expect(item.checksumUrl).toContain(".apk.sha256");
});

it("classifies CUDA and OpenCL miner packages as experimental", () => {
  const minerRelease = { ...release, tag_name: "test-mining-v0.1.0", name: "Sudharma Public Test Mining — Khushi Algorithm v0.1" };
  const cuda = classifyAsset(minerRelease, { name: "khushi-miner-nvidia-windows.zip", size: 1, digest: "sha256:abc", browser_download_url: "https://github.com/sudharma-networks/sudharma/releases/download/test-mining-v0.1.0/khushi-miner-nvidia-windows.zip" });
  const opencl = classifyAsset(minerRelease, { name: "khushi-miner-opencl-windows.zip", size: 1, digest: "sha256:def", browser_download_url: "https://github.com/sudharma-networks/sudharma/releases/download/test-mining-v0.1.0/khushi-miner-opencl-windows.zip" });
  expect(cuda?.slot).toBe("nvidia-miner");
  expect(opencl?.slot).toBe("amd-miner");
  expect(cuda?.channel).toBe("experimental");
  expect(opencl?.channel).toBe("experimental");
});

it("does not promote unknown binary assets", () => {
  const unknown = classifyAsset(release, { name: "mystery.bin", size: 1, browser_download_url: "https://github.com/sudharma-networks/sudharma/releases/download/x/mystery.bin" });
  expect(unknown).toBeNull();
});
