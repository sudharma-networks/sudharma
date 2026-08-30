"use client";

import { useEffect, useState } from "react";
import { ExplorerTransactionDetail } from "@/components/explorer-details";
import { resolveExplorerAPIBaseURL } from "@/lib/explorer-config";

export default function ExplorerTransactionPage() {
  const [transactionId, setTransactionId] = useState("");
  useEffect(() => {
    setTransactionId(new URLSearchParams(window.location.search).get("id") ?? "");
  }, []);
  return (
    <div className="section-shell page-stack explorer-detail-page">
      <ExplorerTransactionDetail apiBaseUrl={resolveExplorerAPIBaseURL(process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL)} transactionId={transactionId} />
    </div>
  );
}
