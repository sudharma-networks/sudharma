# Sudharma Blockchain Explorer v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a real-time read-only Sudharma Public Testnet blockchain explorer inside the existing static website, backed by current canonical node data rather than GitHub or fabricated counters.

**Architecture:** Add canonical read helpers and a bounded `/v1/explorer/*` HTTP namespace to the existing Go node, then upgrade the statically exported Next.js `/explorer` experience to fetch that HTTPS API at runtime. Keep raw RPC loopback-only and publish only the explorer namespace through a dedicated rate-limited reverse proxy.

**Tech Stack:** Go HTTP/RPC + blockchain packages; Next.js App Router static export; TypeScript/React; nginx reverse proxy; GitHub Actions CI.

**Spec:** `docs/superpowers/specs/2026-08-29-blockchain-explorer-v1-design.md`

## Global Constraints

- GitHub versions code/documentation; live explorer data comes from the running Sudharma testnet node.
- Explorer public API is GET-only and exposes no transaction/mining/admin mutation endpoint.
- Raw RPC remains private/loopback; public proxy exposes only `/v1/explorer/`.
- No fabricated counters or fallback demo data.
- Static website detail views use query parameters.
- No mainnet or GPU-PoW activation/deployment changes.

---

### Task 1: Canonical explorer chain reads

**Files:**
- Create: `blockchain/explorer.go`
- Create: `blockchain/explorer_test.go`

**Interfaces:**
- Produces: `ConfirmedTransaction`, `BlockByHash`, `RecentBlocks`, `RecentTransactions`, `TransactionsForAddress`.

- [ ] **Step 1: Write failing tests** proving lowercase 64-hex block lookup, newest-first bounded block pagination, transaction pagination with block metadata, and address history.
- [ ] **Step 2: Run `go test ./blockchain -run 'Explorer|BlockByHash|RecentBlocks|RecentTransactions|TransactionsForAddress' -count=1 -v` and verify RED because the explorer interfaces do not exist.**
- [ ] **Step 3: Implement minimal lock-safe canonical traversal in `blockchain/explorer.go`; reject invalid limits by returning empty results and never mutate chain state.**
- [ ] **Step 4: Re-run the focused test command and verify GREEN.**
- [ ] **Step 5: Commit `feat(explorer): add canonical chain read model`.**

### Task 2: Public read-only explorer HTTP API

**Files:**
- Create: `rpc/explorer.go`
- Create: `rpc/explorer_test.go`
- Modify: `rpc/server.go`
- Modify: `docs/rpc.md`

**Interfaces:**
- Consumes: Task 1 canonical read helpers, existing node mempool/state/status and transaction-status semantics.
- Produces: `/v1/explorer/status`, blocks, transactions, address history, and search endpoints.

- [ ] **Step 1: Write failing handler tests** for status, recent blocks, block lookup by height/hash, pending/confirmed transaction detail, address balance/history, deterministic search resolution, method rejection, malformed input and page-limit caps.
- [ ] **Step 2: Run `go test ./rpc -run Explorer -count=1 -v` and verify RED because routes/handlers are absent.**
- [ ] **Step 3: Implement `rpc/explorer.go` and register `/v1/explorer/` routes in `Server.Handler()`. Use only GET, strict parsing, default page limit 20, maximum 100, and current confirmation calculations.**
- [ ] **Step 4: Update `docs/rpc.md` with the explorer namespace and its public/read-only boundary.**
- [ ] **Step 5: Run `go test ./rpc -run Explorer -count=1 -v`, then `go test ./... -count=1`; verify GREEN.**
- [ ] **Step 6: Commit `feat(explorer): expose bounded read-only chain API`.**

### Task 3: Public reverse-proxy boundary

**Files:**
- Create: `deployment/testnet/nginx-explorer.example.conf`
- Create: `deployment/testnet/explorer-proxy-contract_test.go` or equivalent repository contract test in an existing Go test package.

**Interfaces:**
- Consumes: Task 2 `/v1/explorer/` API.
- Produces: HTTPS proxy template exposing only explorer reads from raw RPC.

- [ ] **Step 1: Write a failing source/config contract test** that requires proxying only `/v1/explorer/`, rejects `/metrics`, `/v1/transactions` submission and mining/admin paths, and requires rate limiting plus HTTPS headers/CORS policy.
- [ ] **Step 2: Run the focused contract test and verify RED because the explorer proxy template is absent.**
- [ ] **Step 3: Add the nginx template with a dedicated rate-limit zone, GET/HEAD/OPTIONS allowance, CORS for the official site origin placeholder, and `proxy_pass http://127.0.0.1:28545/v1/explorer/`; deny all non-explorer locations in that server block.**
- [ ] **Step 4: Re-run the focused contract test and verify GREEN.**
- [ ] **Step 5: Commit `deploy(explorer): define public read-only proxy boundary`.**

### Task 4: Explorer frontend runtime client and dashboard

**Files:**
- Create: `web/lib/explorer-api.ts`
- Create: `web/components/explorer-search.tsx`
- Create: `web/components/explorer-dashboard.tsx`
- Modify: `web/app/explorer/page.tsx`
- Modify: `web/app/globals.css`
- Modify/Add web tests or static source-contract checks according to the repository's existing web test pattern.

**Interfaces:**
- Consumes: Task 2 public API via `NEXT_PUBLIC_EXPLORER_API_URL`.
- Produces: live dashboard, automatic refresh, search navigation and honest unavailable state.

- [ ] **Step 1: Write failing frontend/source contract tests** proving placeholder copy is gone, API URL is externalized, no demo chain values are hard-coded, and search/dashboard components exist.
- [ ] **Step 2: Run repository web checks and verify RED.**
- [ ] **Step 3: Implement typed fetch helpers with request timeout/error handling and no fake fallback data.**
- [ ] **Step 4: Implement search and dashboard components with status/latest blocks/latest transactions, periodic refresh and manual refresh.**
- [ ] **Step 5: Update explorer styling using the existing site design system and mobile responsive rules.**
- [ ] **Step 6: Run web lint/build/static-export checks and verify GREEN.**
- [ ] **Step 7: Commit `feat(web): make explorer dashboard live`.**

### Task 5: Static explorer detail views

**Files:**
- Create: `web/app/explorer/block/page.tsx`
- Create: `web/app/explorer/tx/page.tsx`
- Create: `web/app/explorer/address/page.tsx`
- Create reusable detail components as needed under `web/components/`.
- Update web/source-contract tests.

**Interfaces:**
- Consumes: Task 4 explorer API client.
- Produces: shareable static query-parameter routes for block, transaction and address results.

- [ ] **Step 1: Write failing tests/source checks** requiring all three routes and success/loading/not-found/error behavior.
- [ ] **Step 2: Verify RED.**
- [ ] **Step 3: Implement block details including transactions and canonical hash metadata.**
- [ ] **Step 4: Implement transaction status details including pending/confirmed badge, confirmations and block linkage.**
- [ ] **Step 5: Implement address balance/nonce plus confirmed history with pagination.**
- [ ] **Step 6: Run web lint/build/static export and verify GREEN.**
- [ ] **Step 7: Commit `feat(web): add explorer block transaction and address views`.**

### Task 6: End-to-end verification and integration checkpoint

**Files:**
- Update: explorer plan checkboxes and relevant website documentation.
- Potentially update: `.github/workflows/...` only if existing CI does not exercise required web and Go gates.

**Interfaces:**
- Produces: exact-head CI evidence and deploy-ready explorer build; no public activation until endpoint smoke checks pass.

- [ ] **Step 1: Run focused Go explorer tests, full `go test ./...`, race checks where current CI supports them, and web lint/build/static export.**
- [ ] **Step 2: Inspect exact branch diff to confirm no consensus activation, GPU-PoW activation, raw RPC public exposure, secrets or fake data were introduced.**
- [ ] **Step 3: Verify GitHub Actions exact-head results. Fix failures before proceeding.**
- [ ] **Step 4: Update this plan with exact commit/run evidence.**
- [ ] **Step 5: Reconcile/integrate only after verifying current target branch ancestry; never force-update a diverged long-running branch.**
- [ ] **Step 6: After the HTTPS explorer endpoint exists, smoke-test real `/v1/explorer/status`, block, transaction and search calls before changing the website readiness label from in-development to live.**

## Completion criteria

Explorer v1 is complete only when the live public site can search actual node-backed block/transaction/address data, transaction confirmations reflect the current chain, the static website builds successfully, the public proxy exposes only the read namespace, and exact-head CI is green. If the public HTTPS endpoint is not yet provisioned, code may be integration-ready but the explorer must remain visibly unavailable/in-development rather than pretending to be live.
