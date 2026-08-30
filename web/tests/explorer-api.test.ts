import { expect, test } from "vitest";
import {
  directionLabel,
  formatSUDH,
  shortHash,
  transactionDirection,
} from "@/lib/explorer-api";

test("formats SUDH amounts from base units", () => {
  expect(formatSUDH(100_000_000)).toBe("1 SUDH");
  expect(formatSUDH(50_000_000)).toBe("0.5 SUDH");
});

test("shortens long hashes for display", () => {
  expect(shortHash("abcdef1234567890", 6)).toBe("abcdef…");
});

test("labels address-relative transaction direction", () => {
  const item = {
    transaction: {
      id: "a".repeat(64),
      from: "1".repeat(40),
      to: "2".repeat(40),
      amount: 1,
      fee: 1,
      nonce: 1,
    },
    status: "confirmed",
    confirmations: 1,
  };
  expect(transactionDirection(item, "1".repeat(40))).toBe("sent");
  expect(transactionDirection(item, "2".repeat(40))).toBe("received");
  expect(directionLabel("sent")).toBe("Sent");
});
