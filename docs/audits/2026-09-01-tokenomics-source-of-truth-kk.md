# Sudharma monetary supply source of truth — 2026-09-01

## Owner decision

The final Sudharma mainnet monetary policy is locked at **51,000,000 SUDH (51M)**. The currently running public testnet keeps its separate legacy development policy so its existing chain history is not rewritten.

| Network | Policy constant | Hard cap (SUDH) | Hard cap (atomic) | Purpose |
|---------|-------------------|-----------------|-------------------|---------|
| Public testnet | `MonetaryPolicyPublicTestnet` / `params.MaxSupplySUDH` | **51,000,000,000** (51B) | `51_000_000_000 × 10^8` | Legacy live public-testnet experimentation only |
| Mainnet | `MonetaryPolicyMainnet` / `params.MainnetMaxSupplySUDH` | **51,000,000** (51M) | `51_000_000 × 10^8` | Final mainnet monetary policy; activation remains fail-closed |

These values are intentionally different because the live public testnet predates the final mainnet economics. **51B is not the Sudharma mainnet supply.** All mainnet-facing economics, documentation, readiness checks and future genesis work must use the final 51M policy.

## Final mainnet emission policy

- Maximum supply: **51,000,000 SUDH** exactly.
- Premine: **0**.
- Target block interval: **60 seconds**.
- Main emission span: **5,259,600 subsidy-bearing blocks**.
- Emission schedule: **40 quarterly epochs**, each 131,490 blocks.
- Nominal duration: **10 target years** at the 60-second target interval.
- Block-height selection only; dates and wall-clock time do not alter the reward schedule.
- Mainnet subsidy is permanently **0** after height 5,259,600.
- The deterministic base-unit remainder rule in consensus must make cumulative subsidy equal exactly 51,000,000.00000000 SUDH.

### Annual issuance roadmap

| Target year | Share of cap | SUDH issued | Cumulative SUDH |
|---:|---:|---:|---:|
| 1 | 16% | 8,160,000 | 8,160,000 |
| 2 | 14% | 7,140,000 | 15,300,000 |
| 3 | 13% | 6,630,000 | 21,930,000 |
| 4 | 12% | 6,120,000 | 28,050,000 |
| 5 | 11% | 5,610,000 | 33,660,000 |
| 6 | 10% | 5,100,000 | 38,760,000 |
| 7 | 8% | 4,080,000 | 42,840,000 |
| 8 | 7% | 3,570,000 | 46,410,000 |
| 9 | 5% | 2,550,000 | 48,960,000 |
| 10 | 4% | 2,040,000 | 51,000,000 |

Within each target year the four quarterly targets follow the previously approved 1.03 / 1.01 / 0.99 / 0.97 weighting, encoded in the 40-entry `params.MainnetEmissionEpochs` table. Integer base-unit arithmetic and deterministic remainder blocks keep every epoch and the final 51M cap exact.

## Code source of truth

- Public-testnet legacy cap: `params/params.go` (`MaxSupplySUDH = 51_000_000_000`)
- Final mainnet cap: `params/monetary.go` (`MainnetMaxSupplySUDH = 51_000_000`)
- Final mainnet emission table: `params/mainnet_emission.go`
- Reward selection: `consensus.BlockSubsidyFor(params.MonetaryPolicyMainnet, height)`
- Runtime selection: `params.MonetaryPolicyFor(network)` and `params.MaxSupplyFor(policy)`

Consensus regression tests must continue proving the 40-epoch table totals exactly 51M, the annual issuance targets match this document, and subsidy is zero after height 5,259,600.

## Documentation rule

All current public/operator documentation must describe:

- **51 billion SUDH only as the legacy public-testnet cap**; and
- **51 million SUDH as the final mainnet hard cap and monetary policy**.

Do not label the 51M economics as provisional or as a candidate unless describing historical development context. No document may imply that mainnet inherits the 51B testnet cap.

## Activation remains separate

Finalizing tokenomics does **not** authorize mainnet launch. `MainnetLaunchAuthorized` and `MainnetMiningAuthorized` remain false, and the mainnet genesis timestamp stays unset until the remaining security, physical-GPU, public-review and operator-readiness gates are genuinely complete.

The public testnet remains live under its legacy policy until a separately authorized testnet migration or reset is designed; this source-of-truth does not rewrite existing testnet history.
