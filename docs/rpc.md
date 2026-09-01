# Sudharma Network HTTP RPC API

Step 58 introduces a bounded HTTP JSON API and an RPC-enabled production node binary.

## Run

```bash
go run ./cmd/sudharma-rpcd \
  -nodeid rpc-node-a \
  -listen 127.0.0.1:18444 \
  -rpc 127.0.0.1:18545 \
  -datadir data-rpc-node-a
```

To bootstrap and synchronize from a peer, add `-peer <host:port>`.

The RPC listener defaults to loopback (`127.0.0.1`). Exposing it on a public interface should be done only behind appropriate network controls and TLS termination.

## Endpoints

### `GET /health`

Liveness response:

```json
{"status":"ok"}
```

### `GET /v1/status`

Returns the synchronized node view including node ID, P2P address, advertised height/tip, cumulative work, peer count, mempool count and issued supply.

### `GET /v1/blocks/{height}`

Returns one block by exact height. Invalid heights return `400`; missing heights return `404`.

### `GET /v1/accounts/{address}`

Returns confirmed balance, confirmed nonce and the next expected nonce for the address.

### `GET /v1/mempool?limit=N`

Returns pending transactions. The default page limit is 100 and the maximum is 500.

### `POST /v1/transactions`

Accepts one signed Sudharma `transactions.Transaction` JSON object. The server:

1. verifies transaction identity/signature,
2. rejects already-confirmed or duplicate mempool transactions,
3. validates balance, fee and nonce using the active blockchain state and pending mempool,
4. adds the transaction locally,
5. relays it to connected peers.

Success returns HTTP `202 Accepted` with the transaction ID and number of peers relayed to.

## Transport hardening

The server applies:

- 1 MiB default request-body limit,
- 32 KiB maximum HTTP headers,
- bounded concurrent request handling (128 by default),
- read-header/read/write/idle timeouts,
- strict JSON decoding with unknown-field rejection for transaction submissions,
- method restrictions per route,
- `nosniff` and `no-store` response headers,
- graceful shutdown with a bounded timeout.

These limits are independent of the P2P transport limits added in Step 57.

## Security boundary

The API never accepts private keys and does not sign transactions. Clients must sign locally and submit only the completed signed transaction. This keeps wallet key custody outside the node RPC process.

## Public testnet wallet proxy surfaces

The public HTTPS wallet proxy exposes a reviewed subset of node RPC plus faucet and explorer routes. Live base URL (testnet):

`https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com`

### Explorer (read-only)

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/v1/explorer/status` | Network tip, peers, mempool count, issued supply |
| `GET` | `/v1/explorer/blocks` | Recent blocks |
| `GET` | `/v1/explorer/blocks/{height\|hash}` | Block detail |
| `GET` | `/v1/explorer/transactions` | Recent confirmed transactions |
| `GET` | `/v1/explorer/transactions/{id}` | Transaction detail |
| `GET` | `/v1/explorer/addresses/{address}` | Address balance and history |
| `GET` | `/v1/explorer/mempool` | Pending transactions |
| `GET` | `/v1/explorer/search?q=` | Unified search |

Explorer responses include `Access-Control-Allow-Origin: *` for browser clients.

### Faucet (testnet grants)

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/v1/faucet/info` | Enabled flag, grant sizes, challenge parameters |
| `GET` | `/v1/faucet/health` | Readiness (`ready: true\|false`) |
| `POST` | `/v1/faucet/request` | Initial grant for one address (`{ "address": "<40-hex>" }`) |
| `POST` | `/v1/faucet/challenge` | Challenge reward claim (wallet clients) |

Faucet responses include browser CORS headers. Website clients may POST with `Content-Type: text/plain;charset=UTF-8` and a JSON body to avoid a CORS preflight. Test SUDH has no mainnet value.

### GPU mining (solo, public-testnet)

The public HTTPS proxy and seed nginx allowlists expose GPU mining routes for
`sudharma-gpupow-v1` candidate blocks. This API is independent of the demand
miner (`sudharmad -mineblocks`). CPU and ASIC backends are rejected.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` / `POST` | `/v1/mining/work` | Fetch a GPU candidate block for a wallet address |
| `POST` | `/v1/mining/submit` | Submit a solved block for acceptance and broadcast |

#### `GET` / `POST /v1/mining/work`

Provide the miner's 40-character lowercase hex reward address:

- Query: `?address=<40-hex>`
- JSON body: `{ "address": "<40-hex>" }`

Success returns HTTP `200` with fields including:

- `algorithm`: `sudharma-gpupow-v1`
- `height`, `parent`, `difficulty`, `target`, `timestamp`
- `block`: full candidate block JSON for the GPU hasher
- `pow_compat`: RVN/BTC `getblocktemplate` and ETH `eth_getWork` field aliases

#### `POST /v1/mining/submit`

Accepts the solved candidate block JSON (same shape as the `block` field from
`/v1/mining/work`). On success the node validates PoW, credits
`reward_address`, accepts the block locally, and relays it to peers.

Pool operators use the same HTTP endpoints internally; workers connect to
Stratum pools with `sudharma-miner --stratum stratum+tcp://HOST:3333`. See
`docs/audits/2026-08-31-pool-mining-architecture.md` and
`docs/audits/2026-08-31-mainnet-gpu-mining-architecture.md`.

Live testnet probe:

```bash
curl -fsS -X POST "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com/v1/mining/work" \
  -H 'content-type: application/json' \
  --data '{"address":"YOUR_WALLET_ADDRESS"}'
```

Mainnet GPU mining remains gated until `MainnetMiningAuthorized` is set in a
dedicated activation PR.
