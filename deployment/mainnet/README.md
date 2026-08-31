# Sudharma Mainnet deployment package (draft)

Operator templates for mainnet seed nodes and GPU miners. **Mainnet is not authorized.** Do not deploy until `MainnetLaunchAuthorized` and human launch gates in `docs/audits/2026-08-31-mainnet-launch-operator-runbook.md` are satisfied.

## Topology (planned)

- Two or more independent mainnet seed nodes (`sudharma-mainnet-1` P2P identity)
- P2P TCP `28444` on the public Internet
- Raw `sudharma-rpcd` on loopback `:28545` only
- Public HTTPS / VPC nginx on `:29100` with an explicit route allowlist (mirror public-testnet)
- GPU miners use the same HTTP work/submit model as testnet (`POST /v1/mining/work`, `POST /v1/mining/submit`)

## Files

- `seed1.node.example.json` / `seed2.node.example.json` — replace domain placeholders before deployment
- `public-profile.example.json` — mainnet manifest input; fails preflight until real endpoints exist
- `nginx-rpc.example.conf` — HTTPS reverse proxy starting point with mining routes
- `docker-compose.example.yml` — one-node container with loopback-only raw RPC
- `sudharma-mainnet.service` — hardened systemd unit for native seed installs
- `gpu-miner*.example.json` — GPU miner configs (placeholder RPC URLs until topology publish)
- `deployment-evidence.template.json` — private operator evidence schema (no secrets in git)

## Preflight

After real hostnames are chosen, copy templates outside the repository and run:

```bash
bash ./scripts/testnet-deploy-preflight.sh /secure/path/public-profile.json /secure/path/node.json
```

Mainnet-specific monetary and launch gates:

```bash
go run ./cmd/sudharma-mainnet-readiness
go run ./cmd/sudharma-mainnet-genesis-info
bash ./scripts/mainnet-monetary-rehearsal.sh
bash ./scripts/check-mainnet-readiness-contract_test.sh
bash ./scripts/check-mainnet-go-live-readiness_test.sh
```

## GPU mining (post-launch only)

Mainnet GPU mining stays closed until `params.MainnetMiningAuthorized = true` in a dedicated activation PR. The miner client and Khushi algorithm are the same as public-testnet; only network identity, seeds, and emission policy differ. See `docs/audits/2026-08-31-mainnet-gpu-mining-architecture.md`.
