# Sudharma Blockchain Explorer v1 Design

Date: 2026-08-29
Status: Approved architecture for implementation
Base website branch: `feature/website-foundation`

## Goal

Publish a real-time Sudharma Public Testnet blockchain explorer inside the existing Sudharma website. GitHub remains the source of truth for code and documentation only; live explorer results are read from the running Sudharma node/testnet through a deliberately public, read-only HTTPS API boundary.

## User experience

The existing `/explorer` route becomes a live dashboard instead of a placeholder. Users can search a transaction ID, block height/hash, or wallet address and immediately see the latest canonical node view.

The explorer shows:

- testnet/network status;
- current block height and tip hash;
- latest blocks;
- latest confirmed transactions;
- transaction status (`pending` or `confirmed`), block height/hash and confirmations;
- block details and contained transactions;
- address balance/nonce and confirmed transaction history;
- clear unavailable/error states instead of fabricated data.

Because the website is exported statically, shareable detail views use static routes with query parameters:

- `/explorer/block?id=<height-or-hash>`
- `/explorer/tx?id=<transaction-id>`
- `/explorer/address?address=<wallet-address>`

## Architecture

### Live data source

The browser does not read chain data from GitHub. It calls a public HTTPS explorer API backed by the running Sudharma testnet node. The API derives results from the node's canonical in-memory chain, state and mempool.

The first version does not introduce a database/indexer. Canonical-chain scans are acceptable for current testnet scale and have the important property that reorgs are reflected immediately from the node's active chain. A persistent indexer can be introduced later when chain volume justifies it.

### API namespace

Public explorer reads live under `/v1/explorer/*`:

- `GET /v1/explorer/status`
- `GET /v1/explorer/blocks?limit=N&before=H`
- `GET /v1/explorer/blocks/{height-or-hash}`
- `GET /v1/explorer/transactions?limit=N&before_height=H`
- `GET /v1/explorer/transactions/{txid}`
- `GET /v1/explorer/addresses/{address}`
- `GET /v1/explorer/addresses/{address}/transactions?limit=N&before_height=H`
- `GET /v1/explorer/search?q=...`

The existing `/v1/transactions/{txid}` pending/confirmed semantics remain authoritative and are reused internally rather than redefined.

### Public security boundary

The explorer API is read-only. It must not expose transaction submission, mining work/submission, metrics, administrative controls, secrets, wallet signing, private keys, seed phrases or node-management endpoints.

Public explorer handlers accept GET only. Inputs are length-bounded and validated. Pagination has conservative default and maximum limits. Existing server concurrency and timeout middleware remains in force.

Deployment keeps the raw RPC listener on loopback. A reverse proxy exposes only `/v1/explorer/` over HTTPS with rate limiting and CORS suitable for the official website origin. This prevents the public website from becoming a general-purpose unrestricted RPC proxy.

## Backend responsibilities

### Canonical lookup helpers

Add focused read helpers to `blockchain.Chain`:

- `BlockByHash(hash string) (*Block, bool)`
- `RecentBlocks(limit int, before *uint64) []*Block`
- `RecentTransactions(limit int, beforeHeight *uint64) []ConfirmedTransaction`
- `TransactionsForAddress(address string, limit int, beforeHeight *uint64) []ConfirmedTransaction`

`ConfirmedTransaction` carries a transaction pointer plus immutable block height/hash/timestamp metadata needed by explorer responses.

These helpers hold the chain read lock while traversing canonical blocks and do not mutate consensus state.

### Explorer HTTP handlers

Add a focused `rpc/explorer.go` rather than growing `rpc/server.go` further. `Server.Handler()` registers the `/v1/explorer/` paths while existing RPC behavior remains unchanged.

Status responses expose only public-safe network fields. Transaction detail first checks mempool, then confirmed chain, and computes confirmations from the current canonical height.

### Search resolution

Search is deterministic:

1. decimal-only query -> try block height;
2. 64 lowercase hex -> try transaction ID first, then block hash;
3. otherwise a syntactically valid wallet address -> address result;
4. no match -> 404.

The response returns a result type and canonical explorer path so the frontend can navigate without guessing.

## Frontend responsibilities

### Runtime API configuration

Add `web/lib/explorer-api.ts`. It reads `NEXT_PUBLIC_EXPLORER_API_URL` at build time with a safe empty default. No private credential is embedded in the browser bundle.

When the endpoint is absent or unreachable, the explorer displays `Live explorer temporarily unavailable` and never substitutes demo numbers.

### Dashboard

Upgrade `web/app/explorer/page.tsx` to a client-capable live dashboard using small reusable explorer components. Fetch public status, recent blocks and recent transactions in parallel, refresh status/list data periodically, and provide manual refresh.

### Search

A search component validates non-empty input, calls `/v1/explorer/search`, then navigates to the static detail route returned by the API.

### Detail routes

Create:

- `web/app/explorer/block/page.tsx`
- `web/app/explorer/tx/page.tsx`
- `web/app/explorer/address/page.tsx`

Each reads query parameters in the browser, requests the relevant public API resource and renders loading, not-found, error and success states.

## Freshness

The explorer is near-real-time rather than GitHub-synchronized. Each browser request reads the current testnet node view. Dashboard data refreshes automatically at a modest interval to avoid unnecessary load, while direct searches always perform a fresh request.

No result is claimed as final beyond its current confirmation count; transaction confirmations are recalculated against current canonical height.

## Testing

Backend TDD covers:

- block lookup by hash;
- newest-first bounded block pagination;
- confirmed transaction pagination;
- address history;
- pending and confirmed transaction details;
- search classification and 404 behavior;
- GET-only and malformed-input rejection;
- maximum page limits;
- no mutation endpoints under `/v1/explorer/`.

Frontend checks cover:

- explorer route no longer contains placeholder copy;
- runtime endpoint configuration;
- search result routing;
- API failure renders unavailable/error state rather than fake values;
- static export build succeeds with all explorer routes.

Repository CI and web build/lint gates must be green before integration.

## Deployment

Add an explorer-specific reverse-proxy example that exposes only `/v1/explorer/` to the Internet and keeps raw RPC private. Production DNS/TLS values remain operator-provisioned and no credentials are committed.

The website deployment receives the public explorer API URL as a non-secret build variable/config value. The explorer should become publicly visible only after the HTTPS endpoint is reachable and the live testnet responses pass smoke checks.

## Safety boundaries

- No mainnet activation.
- No GPU-PoW activation or Seed consensus change as part of explorer work.
- No public exposure of raw node RPC.
- No browser access to private keys, seed phrases, credentials, metrics or admin/mining mutation endpoints.
- No fabricated chain counters.
- GitHub hosts/version-controls source; it is not the live chain-data source.
