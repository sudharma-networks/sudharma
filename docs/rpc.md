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
