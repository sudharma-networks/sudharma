# Testnet go-live operator completion record

**Recorded:** 2026-08-31  
**Branch:** `cursor/canonical-integration-guard-8441`  
**RC commit at completion:** `8947c25bf2f968aeaa7ef42427b25c97b41d732c`

This document summarizes operator-gated deploy outcomes. Private artifact digests belong
in the operator evidence vault (`deployment-evidence.template.json` copy), not in git.

## Completed workflow steps

| Step | Workflow | Outcome |
| --- | --- | --- |
| 1 | Explorer Seed RPC Deploy | Both seeds upgraded (`sudharma.service`) |
| 2 | Testnet Public RPC | Lambda deploy with rollback protection |
| 4 | provision-website-visitor-counter | AWS provision + endpoint config commit |
| 6 | Faucet Enable Public | Public faucet enabled on live Lambda |

## Deferred by operator

| Step | Component | Notes |
| --- | --- | --- |
| 5 | Website publish | Static site deploy deferred; visitor counter API provisioned |
| 7 | Android APK release | Wallet APK publish deferred |

Demand miner auto-deploy may report **skipped** when chain work is not pending. That is
acceptable; mark `demand_miner_seed*` as `deferred: true` in private evidence when no
binary promotion occurred.

## Live public endpoints (read-only smoke 2026-08-31)

| Service | URL |
| --- | --- |
| Public RPC | `https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com` |
| Visitor counter API | `https://b8dr97u4ob.execute-api.ap-south-1.amazonaws.com` |

Smoke at height **455** reported `network: sudharma`, explorer data sources
`seed-1`, `seed-2`, `mempool`, `demand-miner`, faucet health ready, visitor counter
responding.

## Operator evidence next steps

1. Copy `deployment/testnet/deployment-evidence.template.json` to a private path.

2. Fill seed/Lambda digests from deploy workflow logs.

3. Set deferred components:

   ```json
   "website": {
     "deferred": true,
     "notes": "Website static publish deferred post core testnet go-live."
   },
   "android_wallet": {
     "deferred": true,
     "notes": "Android APK release deferred post core testnet go-live."
   }
   ```

4. Merge smoke:

   ```bash
   node ./scripts/collect-testnet-deployment-evidence.mjs > /tmp/public-rpc-smoke.json
   ```

5. Verify:

   ```bash
   bash ./scripts/verify-testnet-deployment-evidence.sh /path/to/evidence.json 8947c25bf2f968aeaa7ef42427b25c97b41d732c
   ```

## Infrastructure fixes applied during go-live

- GitHub OIDC trust policy updated for immutable `sub` claims on repos created after
  2026-07-15 (`repo:sudharma-networks@320107455/sudharma@1343485458:*`).
- S3 bucket `sudharma-testnet-demand-miner-981626123397` created in `ap-south-1`.
- Workflow fixes on this branch for S3 publish output, presigned URL handoff, SSM env
  wrapping, and `sudharma.service` restart during seed RPC upgrades.
