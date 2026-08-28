# Sudharma Public Testnet 1 deployment package

This directory contains operator templates for deploying independent Sudharma testnet nodes. It intentionally contains no private keys, wallet passwords, seed phrases, cloud API keys or TLS private keys.

## Recommended topology

Run at least two seed nodes on independent public servers. Expose P2P TCP port `28444` to the Internet. Keep raw RPC bound to the host loopback interface through the container port mapping and expose user/mobile RPC only through an HTTPS reverse proxy with rate limiting. Keep `/metrics` restricted to trusted monitoring access where practical.

The container runs as an unprivileged `sudharma` user, drops Linux capabilities in the Compose template and stores blockchain data in a persistent volume.

## Files

- `Dockerfile`: reproducible node/manifest binaries in a non-root runtime image.
- `docker-compose.example.yml`: one-node container service with loopback-only raw RPC.
- `seed1.node.example.json` / `seed2.node.example.json`: replace the domain placeholders before deployment.
- `public-profile.example.json`: public client/seed manifest input; it intentionally fails launch preflight until real public endpoints replace placeholders.
- `sudharma-testnet.service`: hardened systemd alternative for a native binary installation.
- `nginx-rpc.example.conf`: HTTPS/rate-limited RPC reverse-proxy starting point.

## Preflight

After real hostnames are known, create deployment copies rather than modifying secret material into the repository. Then run:

```bash
bash ./scripts/testnet-deploy-preflight.sh /secure/path/public-profile.json /secure/path/node.json
```

The preflight rejects unresolved placeholders, obvious non-public seeds, malformed node configuration and wallet-secret fields. It runs focused Go tests/builds and emits the deterministic public launch manifest.

## Firewall intent

For a public seed node, permit inbound TCP `28444` from the Internet. Permit SSH only according to the infrastructure operator's administrative policy. Do not expose raw `28545` directly to the Internet when using the supplied Compose layout; the host mapping is loopback-only. Public HTTPS RPC should be exposed on `443` through a reverse proxy. Monitoring endpoints should be access-controlled at the network/proxy layer.

## Data and backup

Node blockchain/state/mempool data live under `/var/lib/sudharma` (or the configured data directory). Node data is operationally recoverable from peers and is not equivalent to wallet custody material. Treasury/user encrypted wallet files and recovery secrets must be backed up separately and must never be copied into node configuration or container images.

## Owner boundary

Actual server purchase, hosting account authorization, DNS ownership, TLS issuance and firewall changes require an authorized operator/owner. The repository stops short of embedding those credentials. Once the two real server addresses and chosen domain/DNS access exist, the templates can be instantiated and the public testnet can be launched and monitored.

## Demand miner (one host only)

The demand miner is an optional service for one selected public-testnet host. It observes the node's loopback-only status endpoint and invokes the bounded native miner only when work is pending. The example configuration uses `http://127.0.0.1:28545`; do not expose that raw status/RPC endpoint to the Internet.

Build both binaries, then install them with the service disabled:

```bash
go build -trimpath -o sudharma-demand-miner ./cmd/sudharma-demand-miner
go build -trimpath -o sudharmad ./cmd/sudharmad
sudo bash deployment/testnet/install-demand-miner.sh
```

Before enabling it, dry-observe the node and validate that its public-testnet identity is visible locally:

```bash
curl --fail --silent http://127.0.0.1:28545/v1/status
```

When the local status is correct, opt in explicitly:

```bash
sudo bash deployment/testnet/install-demand-miner.sh --enable
# Equivalent after a disabled install:
sudo systemctl enable --now sudharma-demand-miner.service
```

Inspect service state and logs with:

```bash
systemctl status sudharma-demand-miner.service
journalctl -u sudharma-demand-miner.service --since today
```

To stop/remove the service assets, preserving both the demand-miner data directory and the node data directory:

```bash
sudo bash deployment/testnet/install-demand-miner.sh --rollback
```

The configured reward address is public and contains no key material, wallet password, seed phrase, or other credential. Do not add private wallet material to the configuration or service environment.
