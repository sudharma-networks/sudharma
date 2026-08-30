import { expect, test } from "vitest";
import { PUBLIC_EXPLORER_API_BASE_URL, resolveExplorerAPIBaseURL } from "@/lib/explorer-config";

test("uses the reviewed public explorer bridge when no build-time override is configured", () => {
  expect(resolveExplorerAPIBaseURL(undefined)).toBe(PUBLIC_EXPLORER_API_BASE_URL);
  expect(PUBLIC_EXPLORER_API_BASE_URL).toBe("https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com");
});

test("keeps NEXT_PUBLIC_EXPLORER_API_BASE_URL as an explicit build-time override", () => {
  expect(resolveExplorerAPIBaseURL(" https://override.example.test/ ")).toBe("https://override.example.test/");
});
