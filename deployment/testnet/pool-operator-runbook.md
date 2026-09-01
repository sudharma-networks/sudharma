# Public testnet pool operator runbook

Deploy the reference Stratum pool operator against live public-testnet mining RPC. This does **not** activate mainnet mining.

## Prerequisites

- Reviewed commit with green CI on pool mining tests
- Dedicated host or seed-adjacent VM with inbound TCP `3333` for Stratum workers
- Pool payout wallet address (40 lowercase hex) — public address only, no private keys on the pool host unless you intentionally operate a hot wallet
- Mining RPC reachable from the pool host (seed nginx `:29100` or public HTTPS proxy)

## 1. Probe mining RPC

```bash
bash ./scripts/probe-testnet-mining-rpc.sh
# Or with explicit URL:
MINING_RPC_URL=https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com \
  POOL_PAYOUT_ADDRESS=YOUR_WALLET bash ./scripts/probe-testnet-mining-rpc.sh
```

Expected: HTTP 200, `algorithm: sudharma-gpupow-v1`, candidate block fields present.

## 2. Configure pool operator

Copy templates outside the repository:

```bash
sudo install -d -m 0750 /etc/sudharma
sudo cp deployment/testnet/pool.example.json /etc/sudharma/pool.json
sudo cp deployment/testnet/pool.env.example /etc/sudharma/pool.env
sudo chmod 0640 /etc/sudharma/pool.json
```

Edit `/etc/sudharma/pool.json`:

- `payout_address` — your pool operator wallet
- `payout_scheme` — `pplns`, `pps`, `solo`, or `fpps`
- `rpc_url` / `rpc_urls` — seed or public proxy URLs with `/v1/mining/*` enabled
- `stratum_listen` — `:3333` or `:3333` on a specific interface

## 3. Build and install (disabled by default)

```bash
go build -trimpath -o ./sudharma-pool ./cmd/sudharma-pool
sudo useradd --system --home /nonexistent --shell /usr/sbin/nologin sudharma-pool 2>/dev/null || true
sudo SUDHARMA_POOL_BIN="$PWD/sudharma-pool" bash ./deployment/testnet/install-pool-operator.sh
```

Local config probe:

```bash
./sudharma-pool -config /etc/sudharma/pool.json -probe
go run ./cmd/sudharma-mining-readiness
```

## 4. Enable (explicit operator action)

```bash
sudo SUDHARMA_POOL_BIN="$PWD/sudharma-pool" bash ./deployment/testnet/install-pool-operator.sh --enable
sudo systemctl status sudharma-pool.service
journalctl -u sudharma-pool.service -n 50 --no-pager
```

## 5. Connect workers

Stratum URL: `stratum+tcp://YOUR_POOL_HOST:3333`

CLI worker:

```bash
go run ./cmd/sudharma-miner \
  --stratum stratum+tcp://YOUR_POOL_HOST:3333 \
  --address YOUR_WALLET \
  --worker rig1
```

Windows: use `Start Pool Mining.bat` from the published GPU miner zip (set pool URL in `gpu-miner-pool.example.json` or env).

Login format: `wallet.worker` (40-hex wallet + worker name).

## 6. Smoke tests

```bash
bash ./scripts/pool-mining-smoke_test.sh
bash ./scripts/probe-testnet-mining-rpc.sh
go run ./cmd/sudharma-pool -config /etc/sudharma/pool.json -probe
```

## Rollback

```bash
sudo systemctl disable --now sudharma-pool.service || true
sudo rm -f /etc/systemd/system/sudharma-pool.service
sudo rm -f /usr/local/bin/sudharma-pool
sudo systemctl daemon-reload
```

## Remote binary upgrade

```bash
export SUDHARMA_POOL_BIN_URL=https://<trusted-url>/sudharma-pool-linux-amd64
bash ./deployment/testnet/remote-install-sudharma-pool-from-url.sh
```

## Security notes

- Never commit wallet private keys or pool hot-wallet seeds
- Rate-limit Stratum and firewall `:3333` to expected worker networks when possible
- Pool operator uses the same HTTP mining API as solo miners; it does not bypass consensus

See `docs/audits/2026-08-31-pool-mining-architecture.md`.
