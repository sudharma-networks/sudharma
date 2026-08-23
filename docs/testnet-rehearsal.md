# Sudharma Public Testnet rehearsal

Step 63 turns the Step 62 configuration into a repeatable pre-launch exercise.

## What is verified

The automated Go rehearsal mines a short source chain, starts two real Sudharma P2P nodes, synchronizes a fresh node, compares chain tip and issued supply, persists chain/state, reloads them from disk, and starts the recovered node again.

The shell rehearsal starts two `sudharma-rpcd` processes using `testnet/rehearsal/node1.json` and `node2.json`, waits for `/ready`, checks both `/v1/status` responses, verifies height/tip/work agreement and verifies that the second node retained a peer. CI runs this rehearsal on every pull request.

Run locally on Linux/macOS/WSL:

```bash
bash ./scripts/testnet-rehearsal.sh
```

The script uses only local loopback ports and disposable `data-testnet-rehearsal` node data. It does not use wallet secrets or cloud credentials.

## Public launch manifest

Public deployment must use a profile containing at least two distinct seed endpoints. Generate the public fingerprint with:

```bash
go run ./cmd/sudharma-testnet-manifest -profile /path/to/public-testnet-profile.json
```

The command refuses to generate a launch manifest until the profile passes the public-launch gate. The resulting JSON records the testnet name, slug, P2P network ID, deterministic genesis hash, public ports and seed endpoints. Wallets, explorers and operators can publish/compare this manifest to avoid connecting to the wrong network.

## What is not performed automatically

This rehearsal does not purchase cloud servers, create DNS records, open firewall rules, provision TLS certificates or enter infrastructure credentials. Those operations require an authorized owner/operator account and are intentionally kept out of the repository.

Before public launch we will replace example seed names with the real hosts, run the same rehearsal against the release build, provision at least two independent public seed nodes, put public RPC behind HTTPS/rate limiting, and validate node identity/genesis on every server.
