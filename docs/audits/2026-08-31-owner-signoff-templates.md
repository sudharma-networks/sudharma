# Owner sign-off templates (copy/paste)

**Purpose:** Ready-to-use text for GitHub PR merge comments and private operator evidence.  
**Replace** bracketed placeholders before use.  
**Do not** commit filled-in evidence (dates, signatures, vault paths) to the public repository.

---

## 1. PR #76 merge comment (GitHub)

Paste as a PR comment when approving and merging  
https://github.com/sudharma-networks/sudharma/pull/76

```
## Owner merge sign-off — Mainnet Tokenomics v1 (#76)

I have reviewed PR #76 as project owner.

**Verified:**
- Mainnet monetary policy encodes the approved 51,000,000 SUDH cap and 40-epoch emission table.
- Public-testnet behavior remains unchanged (genesis, P2P ID, live reward schedule untouched).
- No mainnet activation: `MainnetLaunchAuthorized` stays false; no genesis timestamp freeze in this PR.

**Decision:** Approve and merge to `main`.

**Recorded:** [YOUR_NAME] · [YYYY-MM-DD UTC]

**Follow-up:** Rebase or retarget PR #77 onto merged mainnet tokenomics before merging readiness freeze.
```

---

## 2. PR #77 merge comment (GitHub)

Paste after PR #76 is merged  
https://github.com/sudharma-networks/sudharma/pull/77

```
## Owner merge sign-off — Mainnet readiness freeze (#77)

I have reviewed PR #77 as project owner. PR #76 is merged.

**Verified:**
- `launch_authorized: false` and `launch_ready: false` (mainnet not authorized).
- `mining_stack_ready: true` for public-testnet solo + pool engineering; `mainnet_mining_authorized: false`.
- Isolated mainnet identity and genesis candidate; policy-bound state minting (`NewStateFor`).
- No forbidden activation flags in params; live testnet/AWS auto-deploy unchanged.

**Decision:** Approve and merge (stacked on mainnet tokenomics branch / into `main` per merge plan).

**Recorded:** [YOUR_NAME] · [YYYY-MM-DD UTC]

**Follow-up:** Proceed to independent security audit and genesis timestamp freeze — not activation.
```

---

## 3. Private evidence — post-merge record (vault only)

Store in private evidence vault after #76 and #77 merge. Do **not** commit secrets or this filled form to git.

```json
{
  "kind": "sudharma-mainnet-engineering-merge-evidence",
  "recorded_at": "REPLACE_WITH_ISO8601_UTC",
  "owner_signoff": "REPLACE_WITH_OWNER_NAME",
  "merged_prs": {
    "tokenomics_v1": {
      "pr": 76,
      "merge_commit_sha": "REPLACE_WITH_SHA",
      "merged_at": "REPLACE_WITH_ISO8601_UTC"
    },
    "mainnet_readiness_freeze": {
      "pr": 77,
      "merge_commit_sha": "REPLACE_WITH_SHA",
      "merged_at": "REPLACE_WITH_ISO8601_UTC"
    }
  },
  "readiness_snapshot_at_merge": {
    "launch_authorized": false,
    "launch_ready": false,
    "mining_stack_ready": true,
    "mainnet_mining_authorized": false,
    "mainnet_genesis_timestamp": 0
  },
  "verification_commands_run": [
    "bash ./scripts/verify-mainnet-merge-readiness.sh all"
  ],
  "notes": "Engineering merge only. Mainnet launch remains gated until audit, genesis freeze, seed topology, and explicit launch decision."
}
```

---

## 4. Launch decision memo (vault only — use after audit + genesis freeze)

Use only when independent security audit is complete, genesis timestamp is frozen in a dedicated PR, and seed topology is reviewed.

```
SUBJECT: Sudharma mainnet launch decision — [HOLD | APPROVE ACTIVATION PR]

To: Core team / operators
From: [YOUR_NAME], project owner
Date: [YYYY-MM-DD UTC]

## Context
Engineering merges #76 (tokenomics) and #77 (readiness freeze) are on main at commit [SHA].
Independent security audit reference: [PRIVATE_VAULT_PATH_OR_AUDIT_ID].
Frozen mainnet genesis timestamp: [UNIX_TIMESTAMP] (PR [NUMBER], commit [SHA]).
Mainnet seed topology reviewed: [YES/NO — reference OPERATOR-CHECKLIST completion date].

## Decision
[ ] HOLD — do not open activation PR. Reason: [AUDIT FINDINGS / TOPOLOGY / OTHER]

[ ] APPROVE — authorize a dedicated activation PR that ONLY:
    - Sets MainnetGenesisTimestamp to [FROZEN_UNIX_TIMESTAMP] (if not already merged)
    - Sets MainnetLaunchAuthorized = true
    - Documents seed endpoints and post-deploy smoke commands
    - Optionally sets MainnetMiningAuthorized = true [YES/NO — specify]

## Explicit non-approval
This memo does NOT authorize:
- Automated AWS deploy without workflow_dispatch
- Changes to public-testnet genesis or economics
- GPU mining on mainnet unless MainnetMiningAuthorized is explicitly approved above

## Sign-off
[YOUR_NAME]
[YYYY-MM-DD UTC]
```

---

## 5. Short GitHub review approval (optional one-liner)

**PR #76:**
```
Owner LGTM — tokenomics v1 encoded, testnet unchanged, no mainnet activation. Merge approved.
```

**PR #77:**
```
Owner LGTM — readiness freeze only, all launch gates false, testnet mining stack ready. Merge after #76. No activation.
```

---

## Related docs

- `docs/audits/2026-08-31-pr76-reviewer-summary.md`
- `docs/audits/2026-08-31-pr77-reviewer-summary.md`
- `docs/audits/2026-08-31-mainnet-merge-review-checklist.md`
- `docs/audits/2026-08-31-security-audit-evidence-template.md`
- `docs/audits/2026-08-31-mainnet-genesis-freeze-template.md`
- `deployment/mainnet/OPERATOR-CHECKLIST.md`
