"use client";

import { useState } from "react";

type ExplorerCopyButtonProps = {
  value: string;
  label: string;
};

export function ExplorerCopyButton({ value, label }: ExplorerCopyButtonProps) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      setCopied(false);
    }
  }

  return (
    <button className="button secondary small explorer-copy-button" type="button" onClick={() => void copy()}>
      {copied ? "Copied" : label}
    </button>
  );
}
