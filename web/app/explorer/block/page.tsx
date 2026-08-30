"use client";

import { useEffect, useState } from "react";
import { ExplorerBlockDetail } from "@/components/explorer-details";
import { resolveExplorerAPIBaseURL } from "@/lib/explorer-config";

export default function ExplorerBlockPage() {
  const [blockId, setBlockId] = useState("");
  useEffect(() => {
    setBlockId(new URLSearchParams(window.location.search).get("id") ?? "");
  }, []);
  return (
    <div className="section-shell page-stack explorer-detail-page">
      <ExplorerBlockDetail apiBaseUrl={resolveExplorerAPIBaseURL(process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL)} blockId={blockId} />
    </div>
  );
}
