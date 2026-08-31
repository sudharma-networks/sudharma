# Stage 7 — Public surface operator closure

**Recorded:** 2026-08-31  
**Branch:** `cursor/canonical-integration-guard-8441`

## Goal

Close the remaining public-surface gaps after Stage 6 code landed:

1. Live faucet browser CORS (Lambda redeploy)
2. Amplify website serving Stage 6 faucet UI + approved tokenomics copy
3. Read-only verify scripts for operators

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
- `.github/workflows/promote-website-foundation.yml` (manual-only)

## Out of scope

- Android APK release (still deferred)
- GPU / Mainnet Tokenomics consensus activation
- Automatic AWS mutation
