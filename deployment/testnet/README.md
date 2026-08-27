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
- `demand-miner.example.json`: non-secret, testnet-only demand-miner configuration.
- `sudharma-demand-miner.service`: disabled-by-default hardened supervisor service.
- `install-demand-miner.sh`: idempotent installer with optional explicit activation.
- `install-demand-miner_test.sh`: staged installer and hardening safety checks.

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

## Demand-based public-testnet miner

The demand miner is deliberately separate from the faucet, public RPC proxy, wallet, and consensus code. It reads loopback node status and requests exactly one native `sudharmad -mineblocks 1` operation only when valid transactions are pending. The example reward address `9ccdc094489874bed888ffe4bdf9b8298f4c5131` is a public address only; it contains no private key, seed, wallet file, signing material, or credential.

The native `sudharmad -mineblocks 1` path continues into the normal long-running node loop after mining. The isolated demand-miner runner handles that lifecycle without changing `sudharmad`: it waits for positive pending-transaction evidence and the post-broadcast `Block #... | Transactions: ...` evidence, then terminates and reaps only the unique ephemeral child process it created. Tests cover the real output format, bounded timeout/cancellation behavior and the controlled one-shot exit.

**Activation gate:** keep the supervisor disabled until an authorized operator can perform the staged live-testnet acceptance checks below the repository boundary. CI verification of the runner and packaging is necessary but does not prove host IAM/SSM authority, service installation on a seed, or live block/transaction behavior. Do not weaken child timeouts, kill arbitrary node processes, broaden IAM, or alter consensus behavior to bypass this gate.

### Install disabled

Build the two binaries from the reviewed branch, create the dedicated service account, and install without activation:

```bash
sudo useradd --system --home /nonexistent --shell /usr/sbin/nologin sudharma-miner 2>/dev/null || true
go build -trimpath -o ./sudharma-demand-miner ./cmd/sudharma-demand-miner
go build -trimpath -o ./sudharmad ./cmd/sudharmad
sudo DEMAND_MINER_BIN="$PWD/sudharma-demand-miner" SUDHARMAD_BIN="$PWD/sudharmad" \
  bash ./deployment/testnet/install-demand-miner.sh
```

Installation is idempotent and preserves an existing `/etc/sudharma/demand-miner.json`. Review that file after installation. Raw status stays on `http://127.0.0.1:28545`; do not point the supervisor at the public HTTPS wallet endpoint.

For a no-host-change rehearsal, use a staging root:

```bash
DESTDIR="$(mktemp -d)" \
DEMAND_MINER_BIN="$PWD/sudharma-demand-miner" \
SUDHARMAD_BIN="$PWD/sudharmad" \
bash ./deployment/testnet/install-demand-miner.sh
```

### Observe before activation

Confirm the node is healthy and the service is still disabled:

```bash
curl --fail --silent http://127.0.0.1:28545/ready
curl --fail --silent http://127.0.0.1:28545/v1/status
systemctl is-enabled sudharma-demand-miner.service || true
```

Normal service observation after authorized staged activation is:

```bash
systemctl --no-pager --full status sudharma-demand-miner.service
journalctl -u sudharma-demand-miner.service -n 100 --no-pager
curl --fail --silent http://127.0.0.1:28545/v1/status
```

The supervisor emits structured JSON operational events. It must never log full configuration contents or any wallet secrets.

### Enable gate

Do **not** run the following until CI is green on the reviewed commit, the target seed is explicitly identified, existing deployment authority has been verified without broadening IAM, and the staged live-testnet acceptance procedure is ready. Activation is an explicit operator action, never an installer default:

```bash
sudo DEMAND_MINER_BIN="$PWD/sudharma-demand-miner" SUDHARMAD_BIN="$PWD/sudharmad" \
  bash ./deployment/testnet/install-demand-miner.sh --enable
```

Only one supervisor host should be active initially. Do not enable a second seed host as a fallback without a separate coordination design and review.

### Rollback

Rollback is limited to the isolated supervisor and its own ephemeral data. It must not touch the public-testnet node state:

```bash
sudo systemctl disable --now sudharma-demand-miner.service || true
sudo rm -f /etc/systemd/system/sudharma-demand-miner.service
sudo rm -f /usr/local/bin/sudharma-demand-miner
sudo rm -f /etc/sudharma/demand-miner.json
sudo rm -rf /var/lib/sudharma-demand-miner
sudo systemctl daemon-reload
```

The native `sudharmad` binary is shared operational infrastructure and is intentionally not removed by this rollback sequence.

## Owner boundary

Actual server purchase, hosting account authorization, DNS ownership, TLS issuance and firewall changes require an authorized operator/owner. The repository stops short of embedding those credentials. Once the two real server addresses and chosen domain/DNS access exist, the templates can be instantiated and the public testnet can be launched and monitored.
