# Sudharma monetary supply source of truth — 2026-09-01

## Owner decision

Kk approved the following split between public testnet and mainnet candidate monetary policy:

| Network | Policy constant | Hard cap (SUDH) | Hard cap (atomic) | Purpose |
|---------|-------------------|-----------------|-------------------|---------|
| Public testnet | `MonetaryPolicyPublicTestnet` / `params.MaxSupplySUDH` | **51,000,000,000** (51B) | `51_000_000_000 × 10^8` | Live public testnet experimentation |
| Mainnet candidate | `MonetaryPolicyMainnet` / `params.MainnetMaxSupplySUDH` | **51,000,000** (51M) | `51_000_000 × 10^8` | Fail-closed mainnet candidate until launch authorization |

These are **intentionally different caps**. The public testnet cap must not be described as the mainnet cap, and vice versa.

## Code source of truth

- Public testnet cap: `params/params.go` (`MaxSupplySUDH = 51_000_000_000`)
- Mainnet candidate cap: `params/monetary.go` (`MainnetMaxSupplySUDH = 51_000_000`)
- Runtime selection: `params.MonetaryPolicyFor(network)` and `params.MaxSupplyFor(policy)`

## Documentation rule

Before mainnet genesis freeze, all operator docs, audit materials and public pages must use:

- **“51 billion SUDH public testnet cap”** when referring to the live testnet
- **“51 million SUDH mainnet candidate cap”** when referring to the fail-closed mainnet policy

No document may imply that mainnet inherits the 51B testnet cap.

## Genesis freeze gate

Mainnet genesis timestamp/hash may be frozen only after this split is reflected consistently in code, tests, audit evidence and operator runbooks.
