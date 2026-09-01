# Genesis timestamp freeze — prep checklist (owner, before activation PR)

**Status:** Prep only. **Do not set** `MainnetGenesisTimestamp` until security-review evidence sub-gates close.

Use this after the internal audit stack lands on `main` and the security regression gate is green, and **before** opening the genesis freeze PR.

---

## Decisions you must make (human)

| Decision | Notes |
| --- | --- |
| **Unix timestamp** | Choose one frozen instant (UTC). Example: `1735689600` = 2025-01-01T00:00:00Z — pick deliberately |
| **Public announcement** | How community learns the frozen time (blog, GitHub release, operator runbook) |
| **Seed readiness** | Real hostnames/DNS ready or deferred until post-freeze deploy PR |
| **Mining on mainnet** | Same activation PR or later PR for `MainnetMiningAuthorized` |

---

## Engineering verification (before freeze PR)

```bash
git checkout main
bash ./scripts/pre-audit-engineering-selfcheck.sh
bash ./scripts/security-regression-gate.sh
go run ./cmd/sudharma-mainnet-genesis-info | jq .
bash ./scripts/generate-mainnet-genesis-candidate.sh
go run ./cmd/sudharma-mainnet-readiness | jq .
```

Record genesis candidate hash from `sudharma-mainnet-genesis-info` — after timestamp is set in the freeze PR, hash must match published value.

---

## Allowed changes in genesis freeze PR only

1. Set `params.MainnetGenesisTimestamp` to chosen non-zero value
2. Update docs/runbooks with frozen timestamp and genesis hash
3. **Do not** set `MainnetLaunchAuthorized = true` in the same PR unless launch decision memo approves combined activation

Template: `docs/audits/2026-08-31-mainnet-genesis-freeze-template.md`

---

## Blocked until evidence gates complete

- [ ] `scripts/security-regression-gate.sh` green on frozen candidate
- [ ] Physical GPU evidence gate complete (#24 + checklist)
- [ ] Public/community review window complete
- [ ] Pre-audit selfcheck log archived in private vault

---

## After freeze PR merges (still not full launch)

- Genesis hash is public and immutable
- `MainnetLaunchAuthorized` may still be `false` until launch decision PR
- Operators review `deployment/mainnet/OPERATOR-CHECKLIST.md`
