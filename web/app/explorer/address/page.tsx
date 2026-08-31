"use client";

import { useEffect, useState } from "react";
import { ExplorerAddressDetail } from "@/components/explorer-details";
import { resolveExplorerAPIBaseURL } from "@/lib/explorer-config";

export default function ExplorerAddressPage() {
  const [address, setAddress] = useState("");
  useEffect(() => {
    setAddress(new URLSearchParams(window.location.search).get("address") ?? "");
  }, []);
  return (
    <div className="section-shell page-stack explorer-detail-page">
      <ExplorerAddressDetail apiBaseUrl={resolveExplorerAPIBaseURL(process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL)} address={address} />
    </div>
  );
}
