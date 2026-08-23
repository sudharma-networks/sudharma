# Sudharma Public Testnet 1 launch checklist

This document separates code readiness from infrastructure provisioning. No wallet secret or treasury private key is required to operate seed/RPC nodes.

## Code readiness

- network profile is `sudharma-testnet-1`
- testnet P2P/RPC ports are separated from development/default ports
- deterministic genesis fingerprint is published by `sudharma-testnet-info`
- at least two seed nodes are required before a profile is declared public-launch ready
- node health, readiness and Prometheus metrics are available
- chain/state/mempool persistence is enabled
- graceful shutdown is enabled
- local two-node rehearsal configuration is available
- normal tests and Go race detector must pass before merge

## Infrastructure provisioning

Provision at least two independently hosted seed/full nodes. Prefer different providers or failure domains. Each public seed must expose the P2P port while RPC should remain private unless deliberately published behind HTTPS, firewall/rate limiting and monitoring.

For each node:

1. install the release binary from the same audited commit
2. create a dedicated service account and writable data directory
3. install a node JSON config using the testnet ports/data path
4. expose only the intended P2P port publicly
5. bind raw RPC to loopback/private networking
6. start the service and verify `/health` and `/ready`
7. verify the genesis fingerprint and network metadata match across nodes
8. connect seed nodes to each other and verify peer count/synchronization
9. test restart persistence
10. configure monitoring/alerts for process health, readiness, peer count, chain-height stagnation, persistence errors and disk usage

## Public RPC gateway

A mobile/browser-facing public RPC endpoint should terminate HTTPS outside the Go node, apply request/rate limits, and proxy only the documented RPC routes. Do not expose wallet files or filesystem access through the gateway.

## Mobile release gate

Do not publish the Android network preset until:

- two seed nodes are stable
- one or more HTTPS RPC endpoints are stable
- testnet identity/genesis fingerprint is frozen for that testnet generation
- faucet availability is defined
- explorer availability is defined or the wallet clearly states it is pending

## Manual operator action

Actual cloud/server/DNS provisioning is an operator action because it requires infrastructure accounts, billing and domain control. Repository code intentionally contains no cloud credentials.
