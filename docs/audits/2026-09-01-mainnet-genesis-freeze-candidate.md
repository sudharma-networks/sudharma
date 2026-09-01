# Mainnet genesis freeze candidate — engineering preview

**Status:** Preview only. `params.MainnetGenesisTimestamp` remains `0` until a dedicated owner freeze PR.

## Current unset candidate

Run:

```bash
go run ./cmd/sudharma-mainnet-genesis-info | jq .
bash ./scripts/generate-mainnet-genesis-candidate.sh
```

At commit time on this branch, the unset-timestamp candidate hash is recorded by CI on the land PR head.

## Example timestamp previews

These are **not authorized** until the owner freeze PR merges:

| Timestamp (UTC) | Unix | Example hash (preview) |
| --- | ---: | --- |
| unset | 0 | see `sudharma-mainnet-genesis-info` |
| 2025-01-01T00:00:00Z | 1735689600 | see genesis preview CLI |
| 2027-01-01T00:00:00Z | 1798761600 | see genesis preview CLI |

## Freeze PR requirements

1. All security-review evidence sub-gates except genesis are complete, **or** genesis freeze is explicitly decoupled with written owner sign-off
2. Owner selects one unix timestamp deliberately
3. Set `params.MainnetGenesisTimestamp` only in the freeze PR
4. Publish matching hash from `go run ./cmd/sudharma-mainnet-genesis-info`
5. Keep `MainnetLaunchAuthorized = false` unless the launch decision memo approves combined activation

Template: `docs/audits/2026-08-31-mainnet-genesis-freeze-template.md`
