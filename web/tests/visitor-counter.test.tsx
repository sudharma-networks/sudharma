import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, vi } from "vitest";
import Home from "@/app/page";
import { VisitorCounter } from "@/components/visitor-counter";

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

test("shows a public website visitor counter on the homepage without touching the live service", () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify({ total: 0 }), {
      status: 200,
      headers: { "content-type": "application/json" }
    })
  );
  render(<Home />);
  expect(screen.getByText("Website Visitors")).toBeInTheDocument();
});

test("registers a visit with a simple cross-origin POST that avoids preflight", async () => {
  window.localStorage.setItem("sudharma-visitor-id", "11111111-2222-4333-8444-555555555555");
  const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify({ total: 12 }), {
      status: 200,
      headers: { "content-type": "application/json" }
    })
  );

  render(<VisitorCounter endpoint="https://example.test/v1/website/visitors" />);

  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
  const [, init] = fetchMock.mock.calls[0];
  expect(init?.method).toBe("POST");
  expect(init?.headers).toEqual({ "Content-Type": "text/plain;charset=UTF-8" });
  expect(init?.body).toBe(JSON.stringify({ visitorId: "11111111-2222-4333-8444-555555555555" }));
  expect(await screen.findByText("12")).toBeInTheDocument();
});
