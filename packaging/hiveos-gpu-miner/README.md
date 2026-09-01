# HiveOS GPU miner pack (Sudharma)

Reference flight-sheet settings for Sudharma public-testnet pool or solo mining.

## Solo mining (HTTP work API)

Use the Windows/Linux `sudharma-miner` binary with your wallet address:

```bash
./sudharma-miner -auto -config /path/to/gpu-miner.example.json
```

## Pool mining (Stratum v1)

Point rigs at a Sudharma pool operator running `cmd/sudharma-pool`:

| Field | Value |
| --- | --- |
| Coin | SUDH (custom) |
| Pool URL | `stratum+tcp://YOUR_POOL_HOST:3333` |
| Wallet | your 40-hex Sudharma address |
| Worker | rig name (becomes `wallet.rig`) |
| Password | `x` |

Example CLI on a rig:

```bash
./sudharma-miner \
  -address YOUR_WALLET \
  -stratum stratum+tcp://YOUR_POOL_HOST:3333 \
  -worker rig001
```

Or use `deployment/testnet/gpu-miner-pool.example.json`.

## Notes

- Sudharma is **GPU-only** (`sudharma-gpupow-v1`). CPU and ASIC backends are rejected.
- Mainnet pool mining stays closed until `MainnetMiningAuthorized`.
- See `docs/audits/2026-08-31-pool-mining-architecture.md` for payout schemes (PPS/PPLNS/SOLO/FPPS).
