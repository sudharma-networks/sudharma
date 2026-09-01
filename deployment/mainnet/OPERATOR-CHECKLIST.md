# Mainnet seed topology — operator checklist (draft)

**Mainnet is not authorized.** Use this checklist during topology review only. Do not deploy until `MainnetLaunchAuthorized = true` and human gates in `docs/audits/2026-08-31-mainnet-launch-operator-runbook.md` are satisfied.

## 1. Network identity

| Item | Planned value | Verified |
| --- | --- | --- |
| P2P network ID | `sudharma-mainnet-1` | [ ] |
| Handshake network string | `sudharma` (unchanged) | [ ] |
| P2P port | `28444` | [ ] |
| Raw RPC (loopback) | `127.0.0.1:28545` | [ ] |
| Public HTTPS RPC/nginx | `:29100` with explicit allowlist | [ ] |
| Monetary cap | 51,000,000 SUDH | [ ] |

## 2. Seed hosts (minimum two independent servers)

| Seed | Public P2P | HTTPS RPC hostname | Operator |
| --- | --- | --- | --- |
| seed-1 | `seed1.<domain>:28444` | `https://seed1.<domain>:29100` or nginx front | [ ] |
| seed-2 | `seed2.<domain>:28444` | `https://seed2.<domain>:29100` or nginx front | [ ] |

Copy and customize outside the repository:

- `seed1.node.example.json` → `/secure/path/seed1.node.json`
- `seed2.node.example.json` → `/secure/path/seed2.node.json`
- `public-profile.example.json` → `/secure/path/public-profile.json` (replace `REPLACE_WITH_REAL_DOMAIN`)

## 3. Firewall intent

- [ ] Inbound TCP `28444` from Internet (P2P)
- [ ] Raw RPC `28545` **not** exposed to Internet
- [ ] HTTPS RPC only through nginx with rate limits
- [ ] `/v1/mining/*` allowlisted only when `MainnetMiningAuthorized = true`
- [ ] SSH/admin access restricted to operator policy

## 4. Systemd / container

- [ ] `sudharma-mainnet.service` installed for native seeds, or `docker-compose.example.yml` reviewed
- [ ] Dedicated unprivileged service user
- [ ] Persistent data under `/var/lib/sudharma` (or configured path)
- [ ] Backups: node data recoverable from peers; wallet keys **never** on seed hosts

## 5. Preflight (before first start)

```bash
bash ./scripts/testnet-deploy-preflight.sh /secure/path/public-profile.json /secure/path/seed1.node.json
go run ./cmd/sudharma-mainnet-readiness | jq .
go run ./cmd/sudharma-mainnet-genesis-info | jq .
bash ./scripts/mainnet-monetary-rehearsal.sh
```

Expected before launch authorization:

- `launch_ready: false`
- `launch_authorized: false`
- Genesis timestamp still `0` until freeze PR merges

## 6. Post-activation smoke (after launch PR only)

```bash
curl -fsS http://127.0.0.1:28545/v1/status
curl -fsS https://<public-rpc>/v1/explorer/status
go run ./cmd/sudharma-mainnet-genesis-info | jq -e '.hash == "<published-genesis-hash>"'
```

GPU mining smoke (only when `MainnetMiningAuthorized = true`):

```bash
curl -fsS -X POST "https://<public-rpc>/v1/mining/work" \
  -H 'content-type: application/json' \
  --data '{"address":"<40-hex-wallet>"}'
```

## 7. Evidence

Fill `deployment-evidence.template.json` in a **private vault** (no secrets in git):

- Deploy workflow run IDs
- Genesis hash and timestamp
- Seed peer counts at T+1h
- nginx allowlist confirmation for mining routes (if armed)

## Related docs

- `docs/audits/2026-08-31-mainnet-launch-operator-runbook.md`
- `docs/audits/2026-08-31-mainnet-gpu-mining-architecture.md`
- `docs/audits/2026-08-31-mainnet-genesis-freeze-template.md`
- `docs/audits/2026-08-31-mainnet-merge-review-checklist.md`
