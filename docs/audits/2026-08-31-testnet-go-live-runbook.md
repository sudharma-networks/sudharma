# Testnet go-live operator runbook

**Recorded:** 2026-08-31  
**Applies to branch:** `cursor/canonical-integration-guard-8441`  
**RC attestation:** `docs/audits/2026-08-31-testnet-rc-attestation.md`

This runbook describes **operator-gated** testnet go-live steps. Every mutating action
uses GitHub Actions `workflow_dispatch` only. Nothing in this document authorizes
automatic deployment.

## Preconditions

1. RC attestation passes at the target commit:

   ```bash
   bash ./scripts/generate-testnet-rc-attestation.sh
   bash ./scripts/check-testnet-rc-readiness_test.sh
   ```

2. Main CI is green on the same commit (Go, Lambda, rehearsal, Android, website).

3. Operator has AWS OIDC access for the Sudharma testnet role and seed SSM access.

## Manual deploy sequence

Execute in order. Stop if any step fails rollback or smoke verification.

| Step | Workflow | Purpose |
| --- | --- | --- |
| 1 | `Explorer Seed RPC Deploy` | Publish `sudharma-rpcd` with explorer handlers to both seeds |
| 2 | `Testnet Public RPC` | `preflight: true` first, then `deploy: true` with rollback-protected Lambda promotion |
| 3 | `Demand Miner Auto Deploy` | Ensure demand miners on both seeds match RC binaries |
| 4 | `provision-website-visitor-counter` | Provision visitor counter table/Lambda if not already present |
| 5 | Website publish | Deploy static site build matching RC (outside this repo's auto workflows) |
| 6 | `Faucet Enable Public` | Only after Lambda deploy smoke and seed health pass |
| 7 | Android release | Publish wallet APK only after RPC/faucet/explorer stability is confirmed |

Diagnostics-only or recovery workflows (`faucet-diagnostics-auto-deploy`, prepared payout
recovery, recovery retry) are for incident response — not part of initial go-live.

## Post-deploy evidence collection

1. Copy `deployment/testnet/deployment-evidence.template.json` to a private operator path.

2. Fill component artifact digests from each manual deploy output (seed binary SHA,
   Lambda CodeSha256, demand-miner SHA, website build id, APK checksum).

3. Collect read-only public RPC smoke (does not mutate live services):

   ```bash
   node ./scripts/collect-testnet-deployment-evidence.mjs > /tmp/public-rpc-smoke.json
   ```

   Merge the `public_rpc_smoke` object into the evidence file.

4. Verify evidence against the RC commit:

   ```bash
   bash ./scripts/verify-testnet-deployment-evidence.sh /path/to/evidence.json <RC_COMMIT_SHA>
   ```

5. Record operator sign-off in the evidence file and store it in the operator evidence vault.

## Go-live declaration criteria

Declare testnet go-live readiness only when:

- Evidence file verifies against the RC commit
- Both seeds report matching height/tip on private RPC
- Public RPC smoke shows `network: sudharma` on status and explorer status
- Faucet health is independent of payout enable state until step 6 is deliberately run
- Deployed Lambda CodeSha256 matches the tested artifact from step 2

## Explicit non-actions

- Do not enable faucet payouts during diagnostics-only Lambda rollout
- Do not run GPU staging or mainnet tokenomics workflows for testnet go-live
- Do not merge deployment evidence placeholders into this repository
