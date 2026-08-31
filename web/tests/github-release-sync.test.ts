import { __githubJsonForTests, classifyAsset, isRetryableGitHubStatus, normalizeReleases, withSameSiteWalletUrls } from "../scripts/sync-github-releases.mjs";

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

it("publishes the newest Android wallet through fixed same-site URLs", () => {
  const [wallet] = withSameSiteWalletUrls(normalizeReleases([release]));
  expect(wallet.version).toBe("wallet-testnet-v0.1.0");
  expect(wallet.sha256).toBe("f4d0ec7898bcfd19a857a9930f71a2433c297112e4b1589b6856c1d397d8ebab");
  expect(wallet.downloadUrl).toBe("/downloads/Sudharma-Wallet-latest.apk");
  expect(wallet.checksumUrl).toBe("/downloads/Sudharma-Wallet-latest.apk.sha256");
  expect(wallet.releaseNotesUrl).toBe(release.html_url);
});

it("keeps only the newest verified release for each product slot", () => {
  const olderWallet = {
    ...release,
    tag_name: "android-wallet-v0.1.0-testnet",
    name: "Sudharma Android Wallet 0.1.0 Testnet",
    published_at: "2026-08-26T14:20:19Z",
    html_url: "https://github.com/sudharma-networks/sudharma/releases/tag/android-wallet-v0.1.0-testnet",
    assets: [
      {
        ...release.assets[0],
        browser_download_url: "https://github.com/sudharma-networks/sudharma/releases/download/android-wallet-v0.1.0-testnet/Sudharma-Wallet-0.1.0-testnet.apk"
      }
    ]
  };

  const items = normalizeReleases([olderWallet, release]);
  const wallets = items.filter((item) => item.slot === "android-wallet");

  expect(wallets).toHaveLength(1);
  expect(wallets[0].version).toBe("wallet-testnet-v0.1.0");
});

it("classifies CUDA and OpenCL miner packages as experimental", () => {
  const minerRelease = { ...release, tag_name: "test-mining-v0.1.0", name: "Sudharma Public Test Mining — Khushi Algorithm v0.1" };
  const cuda = classifyAsset(minerRelease, { name: "khushi-miner-nvidia-windows.zip", size: 1, digest: "sha256:abc", browser_download_url: "https://github.com/sudharma-networks/sudharma/releases/download/test-mining-v0.1.0/khushi-miner-nvidia-windows.zip" });
  const opencl = classifyAsset(minerRelease, { name: "khushi-miner-opencl-windows.zip", size: 1, digest: "sha256:def", browser_download_url: "https://github.com/sudharma-networks/sudharma/releases/download/test-mining-v0.1.0/khushi-miner-opencl-windows.zip" });
  expect(cuda?.slot).toBe("nvidia-miner");
  expect(opencl?.slot).toBe("amd-miner");
  expect(cuda?.channel).toBe("experimental");
  expect(opencl?.channel).toBe("experimental");
  expect(cuda).not.toBeNull();

  const [sameSiteCuda] = withSameSiteWalletUrls([cuda!]);
  expect(sameSiteCuda.downloadUrl).toBe(cuda!.downloadUrl);
});

it("does not promote unknown binary assets", () => {
  const unknown = classifyAsset(release, { name: "mystery.bin", size: 1, browser_download_url: "https://github.com/sudharma-networks/sudharma/releases/download/x/mystery.bin" });
  expect(unknown).toBeNull();
});

it("treats rate limits and gateway errors as retryable", () => {
  expect(isRetryableGitHubStatus(403)).toBe(true);
  expect(isRetryableGitHubStatus(429)).toBe(true);
  expect(isRetryableGitHubStatus(503)).toBe(true);
  expect(isRetryableGitHubStatus(404)).toBe(false);
});

it("retries a rate-limited GitHub response before succeeding", async () => {
  const responses = [
    new Response("rate limited", { status: 403 }),
    new Response(JSON.stringify([{ tag_name: "x" }]), { status: 200, headers: { "content-type": "application/json" } })
  ];
  const fetchImpl = ((): typeof fetch => (async () => responses.shift()!) as unknown as typeof fetch)();
  const sleeps: number[] = [];

  const payload = await __githubJsonForTests("/repos/x/y/releases", {
    fetchImpl,
    delayMs: 1,
    sleep: async (ms: number) => { sleeps.push(ms); }
  });

  expect(payload).toEqual([{ tag_name: "x" }]);
  expect(sleeps).toEqual([1]);
});

it("does not retry a non-retryable GitHub failure", async () => {
  let calls = 0;
  const fetchImpl = (async () => { calls += 1; return new Response("nope", { status: 404 }); }) as unknown as typeof fetch;

  await expect(__githubJsonForTests("/repos/x/y/releases", { fetchImpl, delayMs: 1, sleep: async () => {} }))
    .rejects.toThrow(/GitHub API 404/);
  expect(calls).toBe(1);
});
