# Stage 7 — Public surface operator closure

**Recorded:** 2026-08-31  
**Branch:** `cursor/canonical-integration-guard-8441`

## Goal

Close the remaining public-surface gaps after Stage 6 code landed:

1. Live faucet browser CORS (Lambda redeploy)
2. Amplify website serving Stage 6 faucet UI + approved tokenomics copy
3. Read-only verify scripts for operators

## Outcome (2026-08-31)

| Item | Result |
| --- | --- |
| Lambda redeploy (`Testnet Public RPC`, run `33410467973`) | **Done** — rollback snapshot taken, faucet reactivated, smoke passed |
| Faucet browser CORS | **Live** — `node ./scripts/verify-faucet-browser-cors.mjs` reports `PASS` |
| Website promotion to `feature/website-foundation` | **Done** — commit `dbeec63`; Amplify rebuilt |
| Live `/faucet` | **Serving the real request UI** ("Request testnet SUDH safely") |
| Live homepage | Approved tokenomics ("Designed for Scarcity") + `Tokenomics` nav |

### Visitor counter endpoint change

The promotion switched the website from the wallet-proxy path
(`…ja6a03avlc…/v1/website/visitors`) to the dedicated counter provisioned in Stage 5
step 4 (`…b8dr97u4ob…`). The dedicated service keeps website traffic off the wallet
proxy but starts from its own count, so the displayed total restarts. To keep the older
running total instead, set `web/public/data/visitor-counter.json` back to the
`/v1/website/visitors` URL and re-promote.

### Workflow dispatch caveat

`promote-website-foundation.yml` cannot be dispatched with `gh workflow run` until it
exists on the repository default branch (GitHub requirement). Until PR #69 merges, use
the local promote script, which excludes build directories automatically:

```bash
bash ./scripts/promote-website-tree.sh HEAD           # dry run
bash ./scripts/promote-website-tree.sh HEAD --push    # publish
```

## Operator steps (PowerShell)

Use branch `cursor/canonical-integration-guard-8441` for every run.

### A. Redeploy public RPC (CORS)

```powershell
gh workflow run testnet-public-rpc.yml --ref cursor/canonical-integration-guard-8441 -f preflight=false -f deploy=true -f diagnostics_only=false
```

When the run is green:

```powershell
node ./scripts/verify-faucet-browser-cors.mjs
```

Expect: `PASS faucet browser CORS …`

### B. Promote website to Amplify branch

Dry run first:

```powershell
gh workflow run promote-website-foundation.yml --ref cursor/canonical-integration-guard-8441 -f source_ref=cursor/canonical-integration-guard-8441 -f dry_run=true
```

Then promote for real:

```powershell
gh workflow run promote-website-foundation.yml --ref cursor/canonical-integration-guard-8441 -f source_ref=cursor/canonical-integration-guard-8441 -f dry_run=false
```

Amplify rebuilds from `feature/website-foundation`. Confirm `/faucet` no longer shows “In Development”.

## Code on this branch

- Ported approved 51M mainnet tokenomics presentation from `feature/website-foundation`
- `scripts/verify-faucet-browser-cors.mjs`
- `scripts/promote-website-tree.sh` (local promote path, dry-run by default)
- `.github/workflows/promote-website-foundation.yml` (manual-only)

## Out of scope

- Android APK release (still deferred)
- GPU / Mainnet Tokenomics consensus activation
- Automatic AWS mutation
