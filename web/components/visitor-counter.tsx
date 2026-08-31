"use client";

import { useEffect, useState } from "react";

const VISITOR_ID_KEY = "sudharma-visitor-id";

type VisitorCounterProps = {
  endpoint?: string | null;
};

type CounterResponse = {
  total?: number;
};

function visitorId() {
  const existing = window.localStorage.getItem(VISITOR_ID_KEY);
  if (existing) return existing;
  const created = window.crypto.randomUUID();
  window.localStorage.setItem(VISITOR_ID_KEY, created);
  return created;
}

export function VisitorCounter({ endpoint }: VisitorCounterProps) {
  const [count, setCount] = useState<number | null>(null);
  const [available, setAvailable] = useState(Boolean(endpoint));

  useEffect(() => {
    if (!endpoint) return;
    const counterEndpoint = endpoint;
    const controller = new AbortController();

    async function registerVisit() {
      try {
        const response = await fetch(counterEndpoint, {
          method: "POST",
          headers: { "Content-Type": "text/plain;charset=UTF-8" },
          body: JSON.stringify({ visitorId: visitorId() }),
          signal: controller.signal
        });
        if (!response.ok) throw new Error(`visitor counter returned ${response.status}`);
        const body = (await response.json()) as CounterResponse;
        if (typeof body.total !== "number" || !Number.isFinite(body.total) || body.total < 0) throw new Error("invalid visitor counter response");
        setCount(Math.floor(body.total));
        setAvailable(true);
      } catch (error) {
        if ((error as Error)?.name !== "AbortError") setAvailable(false);
      }
    }

    void registerVisit();
    return () => controller.abort();
  }, [endpoint]);

  return (
    <div className="stat" aria-live="polite">
      <span>Website Visitors</span>
      <strong>{count === null ? "—" : count.toLocaleString("en-IN")}</strong>
      <small>{available ? "Approx. unique visits · once per browser/device each day" : "Counter temporarily unavailable"}</small>
    </div>
  );
}
