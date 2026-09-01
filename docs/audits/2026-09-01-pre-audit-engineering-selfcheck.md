# Pre-audit engineering self-check

**Purpose:** Give external auditors (and the owner) a single reproducible baseline before review.  
**Does NOT** close the `independent-security-audit` mainnet gate.

Run after cloning the commit under review:

```bash
bash ./scripts/pre-audit-engineering-selfcheck.sh
```

Optional (skip live network curl):

```bash
SKIP_LIVE_PROBE=1 bash ./scripts/pre-audit-engineering-selfcheck.sh
```

## What it verifies

| Area | Checks |
| --- | --- |
| Go quality | `go vet`, targeted package tests |
| Secrets | `check-tracked-secrets_test.sh` |
| Mainnet gates | readiness CLI: launch blocked, mining stack ready |
| Monetary | `mainnet-monetary-rehearsal.sh` |
| Mining | pool smoke, mining API contract, live RPC probe |
| Forbidden flags | no `MainnetLaunchAuthorized = true` in tree |

## Expected output

Final line:

```json
{"pre_audit_selfcheck":"ok"}
```

Save the full log in your **private evidence vault** alongside audit kickoff records.

## For auditors

Pair this with:

- `docs/audits/2026-09-01-security-audit-scope.md`
- `docs/audits/2026-09-01-security-audit-brief.md`
- `bash ./scripts/verify-mainnet-merge-readiness.sh pr77`

## Owner status (2026-09-01)

Outreach emails sent (Trail of Bits / firms / university path). Awaiting responses. Gate remains open until signed report.
