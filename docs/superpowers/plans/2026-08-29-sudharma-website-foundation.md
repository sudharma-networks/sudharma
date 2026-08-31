# Sudharma Website Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first production-quality Sudharma public website foundation with premium branding, routed information pages, complete Mining/Developer/Wallet hubs, and a trusted Downloads Hub backed by versioned artifact metadata.

**Architecture:** Add a new `web/` application inside the existing Go repository without restructuring blockchain code. Use a statically deployable Next.js/TypeScript frontend with reusable route-level content components, a typed download manifest, and strict separation between public presentation code and later privileged/live backend services. This plan deliberately excludes live Explorer/Testnet/Faucet APIs and report-notification backend logic; those are separate independently testable subsystems and should receive their own plans after the public frontend foundation lands.

**Tech Stack:** Next.js (App Router), TypeScript, React, CSS Modules/global design tokens, Vitest + React Testing Library, Playwright, Node.js 22+, npm, GitHub Actions, AWS Amplify-compatible static/SSR hosting configuration.

**Spec:** `docs/superpowers/specs/2026-08-29-sudharma-website-design.md` plus `docs/superpowers/specs/2026-08-29-sudharma-website-downloads-amendment.md`

## Global Constraints

- Preserve the official logo at `assets/sudharma-logo.png` as the primary brand mark.
- Show `PRE-MAINNET · ACTIVE DEVELOPMENT` globally until project status changes through a reviewed update.
- Never present unfinished functionality as available; use only `Available`, `Testnet`, `Experimental`, `In Development`, or `Planned` labels as appropriate.
- Do not include private keys, wallet seed phrases, AWS credentials, GitHub secrets, privileged RPC credentials, or seed-node administration endpoints in frontend code.
- Public content must not make investment-return, price-growth, or speculative-profit claims.
- The repository license is Apache License 2.0; link to the actual `LICENSE` file and do not replace its terms with an invented summary.
- Active download buttons may exist only for verified artifact URLs present in the typed download manifest.
- Development/testnet binaries must never be visually represented as production-mainnet releases.
- Mobile usability, keyboard accessibility, sufficient contrast, reduced-motion support, and fast Android loading are release requirements.

---

## File Structure

Create or modify the following focused units:

- `web/package.json` — frontend scripts and dependencies.
- `web/tsconfig.json` — TypeScript configuration.
- `web/next.config.ts` — Next.js configuration.
- `web/vitest.config.ts` — unit/component test configuration.
- `web/playwright.config.ts` — browser test configuration.
- `web/app/layout.tsx` — root metadata and global shell.
- `web/app/globals.css` — design tokens, typography, responsive primitives, reduced motion.
- `web/components/site-header.tsx` — desktop/mobile primary navigation.
- `web/components/site-footer.tsx` — official links and readiness statement.
- `web/components/readiness-badge.tsx` — reusable project status indicator.
- `web/components/page-hero.tsx` — shared branded hero layout.
- `web/components/status-chip.tsx` — readiness labels for features/downloads.
- `web/components/ecosystem-card.tsx` — Use/Mine/Build homepage cards.
- `web/components/report-problem-link.tsx` — contextual link into the future Support flow.
- `web/lib/navigation.ts` — canonical route metadata.
- `web/lib/project.ts` — public project constants and pre-mainnet status.
- `web/lib/downloads.ts` — typed download manifest and filtering helpers.
- `web/app/page.tsx` — homepage.
- `web/app/network/page.tsx` — Network page.
- `web/app/sudh/page.tsx` — SUDH page.
- `web/app/wallet/page.tsx` — Wallet hub.
- `web/app/mining/page.tsx` — Mining hub.
- `web/app/mining/[topic]/page.tsx` — detailed mining routes.
- `web/app/developers/page.tsx` — Developer hub.
- `web/app/developers/[topic]/page.tsx` — detailed developer routes.
- `web/app/downloads/page.tsx` — Downloads Hub.
- `web/app/roadmap/page.tsx` — roadmap summary.
- `web/app/docs/page.tsx` — documentation landing page.
- `web/app/community/page.tsx` — official community/contribution links.
- `web/app/support/page.tsx` — support landing page with backend features clearly marked pending until the support subsystem lands.
- `web/app/testnet/page.tsx` — safe informational testnet landing page with no fake live metrics.
- `web/app/explorer/page.tsx` — informational placeholder with explicit readiness state until explorer backend plan lands.
- `web/app/faucet/page.tsx` — informational placeholder with explicit readiness state until faucet integration plan lands.
- `web/public/sudharma-logo.png` — copied from repository official asset for frontend serving.
- `web/tests/` — component/data tests.
- `web/e2e/` — navigation/accessibility smoke tests.
- `.github/workflows/website-ci.yml` — frontend CI.

---

### Task 1: Scaffold the isolated frontend application and test harness

**Files:**
- Create: `web/package.json`
- Create: `web/tsconfig.json`
- Create: `web/next-env.d.ts`
- Create: `web/next.config.ts`
- Create: `web/vitest.config.ts`
- Create: `web/playwright.config.ts`
- Create: `web/tests/setup.ts`
- Create: `web/app/layout.tsx`
- Create: `web/app/page.tsx`
- Create: `web/app/globals.css`

**Interfaces:**
- Consumes: existing repository only; no blockchain package imports.
- Produces: `npm run dev`, `npm run build`, `npm run test`, `npm run test:e2e`, and a root Next.js App Router shell.

- [ ] **Step 1: Write the failing smoke test**

Create `web/tests/home-smoke.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import Home from "@/app/page";

test("renders the Sudharma homepage identity", () => {
  render(<Home />);
  expect(screen.getByRole("heading", { name: /open blockchain/i })).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the test to verify RED**

Run:

```bash
cd web && npm test -- home-smoke.test.tsx
```

Expected: FAIL because the frontend application/test environment does not exist yet.

- [ ] **Step 3: Add minimal Next.js + test configuration**

Use `web/package.json` scripts exactly:

```json
{
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint",
    "typecheck": "tsc --noEmit",
    "test": "vitest run",
    "test:e2e": "playwright test"
  }
}
```

Create the root page with the minimum heading:

```tsx
export default function Home() {
  return <h1>Open Blockchain. Open Development. Built for Everyone.</h1>;
}
```

- [ ] **Step 4: Install dependencies and verify GREEN**

Run:

```bash
cd web
npm install
npm run typecheck
npm test -- home-smoke.test.tsx
npm run build
```

Expected: all commands PASS.

- [ ] **Step 5: Commit**

```bash
git add web
git commit -m "feat(web): scaffold Sudharma website"
```

---

### Task 2: Establish official Sudharma visual system and global shell

**Files:**
- Create: `web/public/sudharma-logo.png`
- Create: `web/components/site-header.tsx`
- Create: `web/components/site-footer.tsx`
- Create: `web/components/readiness-badge.tsx`
- Create: `web/components/page-hero.tsx`
- Create: `web/components/status-chip.tsx`
- Create: `web/lib/navigation.ts`
- Create: `web/lib/project.ts`
- Modify: `web/app/layout.tsx`
- Modify: `web/app/globals.css`
- Test: `web/tests/site-shell.test.tsx`

**Interfaces:**
- Produces: `PRIMARY_NAV`, `PROJECT_STATUS`, `<SiteHeader />`, `<SiteFooter />`, `<ReadinessBadge />`, `<PageHero />`, `<StatusChip />`.
- Consumers: every route created by later tasks.

- [ ] **Step 1: Write shell contract tests**

```tsx
import { render, screen } from "@testing-library/react";
import { SiteHeader } from "@/components/site-header";

it("shows the Downloads route and pre-mainnet status", () => {
  render(<SiteHeader />);
  expect(screen.getByRole("link", { name: "Downloads" })).toHaveAttribute("href", "/downloads");
  expect(screen.getByText(/pre-mainnet/i)).toBeInTheDocument();
});
```

- [ ] **Step 2: Verify RED**

```bash
cd web && npm test -- site-shell.test.tsx
```

Expected: FAIL because the shell components do not exist.

- [ ] **Step 3: Implement canonical navigation and project constants**

`web/lib/navigation.ts` must export:

```ts
export const PRIMARY_NAV = [
  ["Home", "/"],
  ["Network", "/network"],
  ["SUDH", "/sudh"],
  ["Wallet", "/wallet"],
  ["Mining", "/mining"],
  ["Developers", "/developers"],
  ["Downloads", "/downloads"],
  ["Testnet", "/testnet"],
  ["Explorer", "/explorer"],
  ["Faucet", "/faucet"],
  ["Roadmap", "/roadmap"],
  ["Docs", "/docs"],
  ["Community", "/community"],
  ["Support", "/support"]
] as const;
```

`web/lib/project.ts` must export:

```ts
export const PROJECT_STATUS = "PRE-MAINNET · ACTIVE DEVELOPMENT" as const;
export const PROJECT_NAME = "Sudharma Network" as const;
export const COIN_SYMBOL = "SUDH" as const;
```

- [ ] **Step 4: Implement premium responsive shell**

Copy `assets/sudharma-logo.png` byte-for-byte to `web/public/sudharma-logo.png`, render it in header/footer, add dark navy/black design tokens, restrained luminous accents, keyboard focus states, mobile navigation, and `prefers-reduced-motion` overrides.

- [ ] **Step 5: Verify shell tests, typecheck, and build**

```bash
cd web
npm test -- site-shell.test.tsx
npm run typecheck
npm run build
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web
 git commit -m "feat(web): add Sudharma visual system and navigation"
```

---

### Task 3: Build the premium homepage and ecosystem pathways

**Files:**
- Create: `web/components/ecosystem-card.tsx`
- Modify: `web/app/page.tsx`
- Test: `web/tests/home-content.test.tsx`

**Interfaces:**
- Consumes: global shell components and project constants.
- Produces: direct navigation into Use/Wallet, Mine/Mining, Build/Developers, Downloads, and Support.

- [ ] **Step 1: Write homepage content/navigation tests**

```tsx
render(<Home />);
expect(screen.getByRole("link", { name: /mine sudharma/i })).toHaveAttribute("href", "/mining");
expect(screen.getByRole("link", { name: /build on sudharma/i })).toHaveAttribute("href", "/developers");
expect(screen.getByRole("link", { name: /downloads/i })).toHaveAttribute("href", "/downloads");
```

- [ ] **Step 2: Run RED test**

```bash
cd web && npm test -- home-content.test.tsx
```

Expected: FAIL because the ecosystem cards are not implemented.

- [ ] **Step 3: Implement homepage sections**

Render, in order: hero, readiness badge, `Use Sudharma`, `Mine Sudharma`, `Build on Sudharma`, More Than a Coin, current development snapshot with no fabricated live counters, SUDH summary, current-vs-planned capabilities, security/transparency, roadmap preview, Downloads CTA, Community CTA, Support CTA.

- [ ] **Step 4: Verify**

```bash
cd web
npm test -- home-content.test.tsx
npm run typecheck
npm run build
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/app/page.tsx web/components/ecosystem-card.tsx web/tests/home-content.test.tsx
git commit -m "feat(web): build Sudharma premium homepage"
```

---

### Task 4: Build Network and SUDH reference pages

**Files:**
- Create: `web/app/network/page.tsx`
- Create: `web/app/sudh/page.tsx`
- Test: `web/tests/network-sudh.test.tsx`

**Interfaces:**
- Consumes: current documented project parameters and shared status components.
- Produces: static authoritative educational pages with explicit pre-mainnet wording.

- [ ] **Step 1: Write parameter tests**

```tsx
render(<SudhPage />);
expect(screen.getByText("100,000,000 SUDH")).toBeInTheDocument();
expect(screen.getByText("50 SUDH")).toBeInTheDocument();
expect(screen.getByText("60 seconds")).toBeInTheDocument();
expect(screen.getByText(/subject to change before mainnet/i)).toBeInTheDocument();
```

- [ ] **Step 2: Verify RED**

```bash
cd web && npm test -- network-sudh.test.tsx
```

- [ ] **Step 3: Implement Network page**

Cover Proof of Work, block validation, cumulative-work selection, P2P, blocks/transactions, chain sync/reorg, mempool, identity, security hardening, run-a-node entry point, and the visual sequence `Wallet → Transaction → Nodes → Mempool → Miner → Block → Network`.

- [ ] **Step 4: Implement SUDH page**

Render exactly the currently documented development parameters: 8 decimals, 100,000,000 maximum supply, 50 SUDH initial reward, 60-second target, 1,000,000-block halving interval, 0 premine, 0.10% total fee, 0.01% development portion, 0.09% miner portion, with the pre-mainnet warning.

- [ ] **Step 5: Verify and commit**

```bash
cd web && npm test -- network-sudh.test.tsx && npm run typecheck && npm run build
git add web/app/network web/app/sudh web/tests/network-sudh.test.tsx
git commit -m "feat(web): add Network and SUDH reference pages"
```

---

### Task 5: Build Wallet, Mining, and Developer hubs with real routed detail pages

**Files:**
- Create: `web/app/wallet/page.tsx`
- Create: `web/app/mining/page.tsx`
- Create: `web/app/mining/[topic]/page.tsx`
- Create: `web/app/developers/page.tsx`
- Create: `web/app/developers/[topic]/page.tsx`
- Test: `web/tests/hubs.test.tsx`

**Interfaces:**
- Mining topics: `nvidia | amd | solo | pools | kryptex | benchmarks | troubleshooting`.
- Developer topics: `getting-started | node | rpc | wallets | payments | contributing | protocol`.
- Produces: complete detailed content pages rather than nonfunctional cards.

- [ ] **Step 1: Write route/content tests**

```tsx
render(<MiningPage />);
expect(screen.getByRole("link", { name: /nvidia/i })).toHaveAttribute("href", "/mining/nvidia");
expect(screen.getByRole("link", { name: /amd/i })).toHaveAttribute("href", "/mining/amd");
expect(screen.getByRole("link", { name: /kryptex/i })).toHaveAttribute("href", "/mining/kryptex");
```

- [ ] **Step 2: Verify RED**

```bash
cd web && npm test -- hubs.test.tsx
```

- [ ] **Step 3: Implement Wallet hub**

Include Android/CLI availability sections, create/send/receive guidance, backup/security, testnet link, faucet link, verified Downloads CTA, and development-wallet warning.

- [ ] **Step 4: Implement Mining hub and topic routes**

Each topic route must contain useful detailed explanatory content, safe configuration guidance only where repository-backed, links to Downloads, and a readiness label. If a capability is unfinished, show `In Development`, `Experimental`, or `Planned` instead of an active download/start control.

- [ ] **Step 5: Implement Developer hub and topic routes**

Include build-from-source, node, RPC/API, wallet/payment integration, protocol, GitHub/contribution, and a separate planned-capabilities area for SDKs/tokens/smart contracts.

- [ ] **Step 6: Verify and commit**

```bash
cd web && npm test -- hubs.test.tsx && npm run typecheck && npm run build
git add web/app/wallet web/app/mining web/app/developers web/tests/hubs.test.tsx
git commit -m "feat(web): add Wallet Mining and Developer hubs"
```

---

### Task 6: Implement the trusted Downloads Hub and typed artifact manifest

**Files:**
- Create: `web/lib/downloads.ts`
- Create: `web/components/download-card.tsx`
- Create: `web/app/downloads/page.tsx`
- Test: `web/tests/downloads.test.tsx`

**Interfaces:**
- Produces: `DownloadChannel`, `DownloadKind`, `DownloadArtifact`, `DOWNLOADS`, `filterDownloads(kind)`.
- Consumers: Downloads page, Wallet/Mining/Developer links, later release automation.

- [ ] **Step 1: Write manifest safety tests**

```ts
import { DOWNLOADS } from "@/lib/downloads";

it("never exposes a download URL for unavailable artifacts", () => {
  for (const artifact of DOWNLOADS) {
    if (artifact.status !== "available") {
      expect(artifact.downloadUrl).toBeUndefined();
    }
  }
});

it("requires provenance for available artifacts", () => {
  for (const artifact of DOWNLOADS.filter((a) => a.status === "available")) {
    expect(artifact.sourceUrl ?? artifact.releaseNotesUrl).toBeTruthy();
  }
});
```

- [ ] **Step 2: Verify RED**

```bash
cd web && npm test -- downloads.test.tsx
```

- [ ] **Step 3: Implement typed manifest**

Use exactly:

```ts
export type DownloadChannel = "stable" | "testnet" | "experimental" | "development";
export type DownloadKind = "wallet" | "miner" | "node" | "source" | "developer";
export type DownloadStatus = "available" | "in-development" | "planned";

export interface DownloadArtifact {
  id: string;
  kind: DownloadKind;
  name: string;
  version: string;
  channel: DownloadChannel;
  platform: string;
  architecture: string;
  fileSize?: string;
  sha256?: string;
  releaseDate?: string;
  downloadUrl?: string;
  releaseNotesUrl?: string;
  sourceUrl?: string;
  status: DownloadStatus;
}
```

Seed the manifest only with links/statuses verified during implementation. If a binary is not verified, represent it as `in-development` or `planned` with no `downloadUrl`.

- [ ] **Step 4: Build Downloads UI**

Render Wallets, Miners, Node Software, Source Code, and Developer Resources as separately filterable premium sections. Available artifacts show version/platform/checksum metadata and an active download control; unavailable artifacts show only status. Add official repository + Apache-2.0 license links and a download-safety notice.

- [ ] **Step 5: Verify and commit**

```bash
cd web && npm test -- downloads.test.tsx && npm run typecheck && npm run build
git add web/lib/downloads.ts web/components/download-card.tsx web/app/downloads web/tests/downloads.test.tsx
git commit -m "feat(web): add trusted Sudharma Downloads Hub"
```

---

### Task 7: Add Roadmap, Docs, Community, and safe pre-integration service pages

**Files:**
- Create: `web/app/roadmap/page.tsx`
- Create: `web/app/docs/page.tsx`
- Create: `web/app/community/page.tsx`
- Create: `web/app/support/page.tsx`
- Create: `web/app/testnet/page.tsx`
- Create: `web/app/explorer/page.tsx`
- Create: `web/app/faucet/page.tsx`
- Test: `web/tests/readiness-pages.test.tsx`

**Interfaces:**
- Produces: navigable destinations for all primary nav items without claiming unavailable backend functionality.

- [ ] **Step 1: Write readiness assertion tests**

```tsx
render(<ExplorerPage />);
expect(screen.getByText(/in development|planned|testnet/i)).toBeInTheDocument();
expect(screen.queryByText(/live blocks: 12345/i)).not.toBeInTheDocument();
```

- [ ] **Step 2: Verify RED**

```bash
cd web && npm test -- readiness-pages.test.tsx
```

- [ ] **Step 3: Implement static route content**

Roadmap uses only `Completed`, `In Development`, `Planned`. Docs exposes the approved documentation categories. Community exposes only official GitHub/contribution/security destinations. Testnet/Explorer/Faucet show accurate readiness messaging and no fabricated live values. Support explains Report Problem / Track / Known Issues / Security as the intended subsystem and marks functions unavailable until the support backend plan lands.

- [ ] **Step 4: Verify and commit**

```bash
cd web && npm test -- readiness-pages.test.tsx && npm run typecheck && npm run build
git add web/app/roadmap web/app/docs web/app/community web/app/support web/app/testnet web/app/explorer web/app/faucet web/tests/readiness-pages.test.tsx
git commit -m "feat(web): add roadmap docs community and readiness pages"
```

---

### Task 8: Add contextual problem-report links without prematurely implementing the backend

**Files:**
- Create: `web/components/report-problem-link.tsx`
- Modify: `web/app/wallet/page.tsx`
- Modify: `web/app/mining/page.tsx`
- Modify: `web/app/mining/[topic]/page.tsx`
- Modify: `web/app/developers/page.tsx`
- Modify: `web/app/downloads/page.tsx`
- Test: `web/tests/report-links.test.tsx`

**Interfaces:**
- Produces: `ReportProblemLink({ component, context })` that links to `/support?component=...&context=...` using safe public metadata only.
- Later support subsystem consumes those query parameters.

- [ ] **Step 1: Write encoding/safety tests**

```tsx
render(<ReportProblemLink component="Mining" context="NVIDIA" />);
expect(screen.getByRole("link", { name: /report problem/i })).toHaveAttribute(
  "href",
  "/support?component=Mining&context=NVIDIA"
);
```

- [ ] **Step 2: Verify RED**

```bash
cd web && npm test -- report-links.test.tsx
```

- [ ] **Step 3: Implement contextual links**

The component accepts only strings passed explicitly by page code; it must not collect wallet addresses, browser storage, secrets, credentials, or node-admin data.

- [ ] **Step 4: Add contextual links to hubs and Downloads cards**

Each relevant page exposes an obvious report action. Download report links include public artifact ID/version only when those values are already displayed publicly.

- [ ] **Step 5: Verify and commit**

```bash
cd web && npm test -- report-links.test.tsx && npm run typecheck && npm run build
git add web/components/report-problem-link.tsx web/app web/tests/report-links.test.tsx
git commit -m "feat(web): add contextual support entry points"
```

---

### Task 9: Add SEO, sitemap, accessibility, and browser navigation coverage

**Files:**
- Modify: `web/app/layout.tsx`
- Create: `web/app/sitemap.ts`
- Create: `web/app/robots.ts`
- Create: `web/e2e/navigation.spec.ts`
- Create: `web/e2e/accessibility-smoke.spec.ts`

**Interfaces:**
- Produces: discoverable metadata and browser-level route guarantees.

- [ ] **Step 1: Write browser navigation test**

```ts
import { test, expect } from "@playwright/test";

test("Mining and Downloads navigation opens full pages", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("link", { name: "Mining" }).click();
  await expect(page).toHaveURL(/\/mining$/);
  await expect(page.getByRole("heading", { name: /mine sudharma/i })).toBeVisible();
  await page.getByRole("link", { name: "Downloads" }).click();
  await expect(page).toHaveURL(/\/downloads$/);
  await expect(page.getByRole("heading", { name: /downloads/i })).toBeVisible();
});
```

- [ ] **Step 2: Verify RED before final navigation wiring**

```bash
cd web && npm run test:e2e -- navigation.spec.ts
```

- [ ] **Step 3: Add metadata, sitemap, robots, focus states, and accessibility checks**

Every page gets a descriptive title/description; social metadata uses official Sudharma branding. Ensure heading hierarchy, form/link labels, focus visibility, mobile menu keyboard behavior, and reduced-motion behavior.

- [ ] **Step 4: Verify**

```bash
cd web
npm run typecheck
npm test
npm run build
npm run test:e2e
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web
git commit -m "test(web): add navigation accessibility and SEO coverage"
```

---

### Task 10: Add isolated website CI without affecting existing blockchain CI

**Files:**
- Create: `.github/workflows/website-ci.yml`

**Interfaces:**
- Consumes: `web/package-lock.json`, website scripts.
- Produces: frontend checks on PRs and pushes without modifying the existing Go/GPU workflow behavior.

- [ ] **Step 1: Add CI workflow**

Use path filters for `web/**` and the website workflow itself. Required commands:

```yaml
- run: npm ci
  working-directory: web
- run: npm run typecheck
  working-directory: web
- run: npm test
  working-directory: web
- run: npm run build
  working-directory: web
```

Do not add AWS deployment credentials or production deployment in this task.

- [ ] **Step 2: Validate workflow and full local suite**

```bash
cd web
npm ci
npm run typecheck
npm test
npm run build
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/website-ci.yml web/package-lock.json
git commit -m "ci(web): add isolated Sudharma website checks"
```

---

## Follow-On Plans Required After This Foundation

The approved architecture contains multiple independently testable subsystems, so they must not be mixed into this first implementation plan. Create separate plans after this foundation is green:

1. `Sudharma Live Testnet + Explorer + Faucet Web Integration` — public read-only API contracts, safe live metrics, explorer search/details, faucet transaction flow.
2. `Sudharma Support Reporting + Notifications` — secure uploads, report IDs, GitHub synchronization, private vulnerability route, admin dashboard, email/browser notifications, status tracking.
3. `Sudharma Website AWS Deployment` — Amplify/CDN/HTTPS, environment separation, custom domain when available, rollback and deployment verification.
4. `Sudharma Release Metadata Automation` — trusted GitHub Release ingestion, checksum verification, manifest publication, download provenance automation.

## Plan Self-Review

- Spec coverage: public premium UI, routed detailed pages, official branding, pre-mainnet honesty, Wallet/Mining/Developers, Downloads, static Testnet/Explorer/Faucet readiness pages, roadmap/docs/community/support entry points, contextual problem reporting links, SEO/accessibility/performance expectations, and CI are covered.
- Deliberately deferred with explicit follow-on plans: live blockchain APIs, faucet transactions, explorer backend/indexer, report uploads/GitHub writes/notifications/admin authentication, AWS production deployment, and automated release metadata.
- Placeholder scan: no `TBD`, `TODO`, or unbounded implementation instructions are used.
- Type consistency: Downloads types and contextual report-link interface are defined once and consumed consistently.
