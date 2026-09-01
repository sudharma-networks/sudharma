# Independent security audit — evidence template

**Status:** Not started. This template records what reviewers and operators need **after** an external audit. It does not substitute for an audit.

## Scope (recommended)

| Area | Review focus |
| --- | --- |
| Consensus / monetary | 51M cap enforcement, epoch table, fee vs subsidy accounting |
| P2P / sync | Network ID isolation, handshake, reorg handling |
| Wallet / signing | Key custody boundaries, no seed exposure on public surfaces |
| RPC / public proxy | Route allowlists, body limits, no private-key endpoints |
| GPU mining API | Work/submit validation, GPU-only policy, pool share semantics |
| Operator workflows | Manual-only AWS gates, no secret commits |

## Evidence fields (private vault — do not commit secrets)

```json
{
  "kind": "sudharma-mainnet-security-audit",
  "audit_firm": "REPLACE_WITH_AUDITOR",
  "audit_completed_at": "2026-00-00T00:00:00Z",
  "repository_commit_audited": "REPLACE_WITH_SHA",
  "scope": ["consensus", "p2p", "wallet", "rpc", "mining", "deployment"],
  "findings_summary": {
    "critical": 0,
    "high": 0,
    "medium": 0,
    "low": 0,
    "informational": 0
  },
  "remediation_status": "REPLACE_WITH_OPEN_OR_CLOSED",
  "signed_report_reference": "REPLACE_WITH_PRIVATE_VAULT_PATH",
  "mainnet_launch_recommendation": "hold|approve_after_remediation",
  "reviewer_signoff": "REPLACE_WITH_HUMAN_IDENTITY"
}
```

## Gate linkage

Until audit evidence exists:

- `params/readiness.go` gate `independent-security-audit` stays `ready: false`
- `go run ./cmd/sudharma-mainnet-readiness` reports `launch_ready: false`
- Do not open the genesis freeze / activation PR

## Post-audit operator steps

1. Record audit reference in private evidence (not git)
2. Remediate critical/high findings or document accepted risk with sign-off
3. Re-run verification on the remediated commit:

```bash
bash ./scripts/verify-mainnet-merge-readiness.sh pr77
bash ./scripts/mainnet-monetary-rehearsal.sh
```

4. Proceed to genesis timestamp freeze only after written launch decision

## Related

- `docs/audits/2026-08-31-mainnet-genesis-freeze-template.md`
- `deployment/mainnet/deployment-evidence.template.json`
