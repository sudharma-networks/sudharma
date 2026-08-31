import { DOWNLOADS } from "@/lib/downloads";
import { render, screen } from "@testing-library/react";
import { DownloadCard } from "@/components/download-card";

it("never exposes a download URL for unavailable artifacts", () => { for (const artifact of DOWNLOADS) if (artifact.status !== "available") expect(artifact.downloadUrl).toBeUndefined(); });
it("requires provenance for available artifacts", () => { for (const artifact of DOWNLOADS.filter((a) => a.status === "available")) expect(artifact.sourceUrl ?? artifact.releaseNotesUrl).toBeTruthy(); });

it("serves the Android wallet from this website without a GitHub login", () => {
  const wallet = DOWNLOADS.find((artifact) => artifact.kind === "wallet" && artifact.status === "available");
  expect(wallet?.downloadUrl).toBe("/downloads/Sudharma-Wallet-latest.apk");
  render(<DownloadCard artifact={wallet!} />);
  const button = screen.getByRole("link", { name: "Download" });
  expect(button).toHaveAttribute("href", "/downloads/Sudharma-Wallet-latest.apk");
  expect(button).toHaveAttribute("download", "Sudharma-Wallet-latest.apk");
  expect(button).not.toHaveAttribute("target");
});

