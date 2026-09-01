# Network-bound transaction signatures

## Problem

Transaction signatures previously signed only the transaction ID. A valid testnet signature could be replayed on mainnet when the same key had balance and nonce on both networks.

## Domain versions

| Domain | Preimage | Accepted on |
|--------|----------|-------------|
| Legacy (v1) | `txID` | Public testnet only (backward compatibility) |
| Network-bound (v2) | `sudharma-tx-v2\|<network>\|<txID>` | All networks; required on mainnet |

## Activation

- New wallet/CLI/Android signatures use v2 with the active network ID.
- Public testnet continues accepting already-signed v1 transactions.
- Mainnet rejects v1 signatures at consensus verification time.
- Mainnet launch remains fail-closed until all security gates pass.

## Migration

Operators do not need to resign historical confirmed testnet transactions. New submissions should use v2 automatically through updated clients.

For public-testnet rollout, update node/RPC binaries first so `/v1/status` exposes the canonical `network_id`. Only after nodes report that identity should updated CLI/faucet clients be deployed. The CLI fails closed when `network_id` is missing, unsupported, or does not match the operator-selected network, so mixed-version rollout cannot silently sign for the wrong chain.
