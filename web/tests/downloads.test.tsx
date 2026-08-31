import { DOWNLOADS } from "@/lib/downloads";
it("never exposes a download URL for unavailable artifacts", () => { for (const artifact of DOWNLOADS) if (artifact.status !== "available") expect(artifact.downloadUrl).toBeUndefined(); });
it("requires provenance for available artifacts", () => { for (const artifact of DOWNLOADS.filter((a) => a.status === "available")) expect(artifact.sourceUrl ?? artifact.releaseNotesUrl).toBeTruthy(); });
