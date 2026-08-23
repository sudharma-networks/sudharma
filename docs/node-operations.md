# Sudharma production node operations

`cmd/sudharma-rpcd` is the production-oriented full node entry point. It runs the persistent chain/state/mempool, P2P service and bounded HTTP RPC together.

## Configuration

Use `-config node.json`. Command-line flags can override node ID, P2P/RPC addresses, data directory and JSON logging. A production example:

```json
{
  "node_id": "seed-1",
  "p2p_address": "0.0.0.0:18444",
  "rpc_address": "127.0.0.1:18545",
  "peers": ["seed2.example.org:18444"],
  "data_directory": "/var/lib/sudharma",
  "log_json": true,
  "metrics": true,
  "persist_every": "30s"
}
```

Do not put wallet passwords, private keys, recovery material or exchange credentials in this configuration. Sudharma RPC accepts already-signed transactions and does not require custody of user keys.

## Health and monitoring

- `GET /health`: liveness; the HTTP process is serving requests.
- `GET /ready`: readiness; chain/node dependencies have a usable current tip.
- `GET /metrics`: Prometheus text format when metrics are enabled.

Current metrics expose chain height, connected peer count, mempool transaction count and issued native supply. Operators should alert on process/readiness failure, persistence errors, prolonged peer count of zero for public nodes, unexpected chain-height stagnation and abnormal mempool growth.

## Persistence and shutdown

The node periodically persists chain, state and mempool according to `persist_every`. A persistence failure is treated as an operator-visible fatal error rather than silently continuing indefinitely. SIGINT/SIGTERM triggers bounded RPC shutdown, a final persistence pass and P2P shutdown.

Back up the data directory for operational recovery, but do not confuse node data with wallet backups. Wallet encrypted files and owner treasury recovery material require their own offline backup policy.

## Exposure

P2P may be exposed publicly for public/seed nodes. RPC defaults to loopback and should remain private unless placed behind an intentionally configured reverse proxy/firewall/TLS/authentication layer. `/metrics` should likewise be restricted to trusted monitoring networks when RPC is externally proxied.
