# Security audit kickoff — owner checklist

**Owner:** Kk  
**Status:** Kickoff package ready — **external audit not yet commissioned**  
**Repository commit for audit:** `a63fe384a75a8857f02a78b612be5b9b0a233cb8` (main after #89)  
**Gate:** `independent-security-audit` stays `ready: false` until signed report is recorded

Engineering merges (#76, #77, #89) are complete. Mainnet launch remains blocked until this gate closes.

---

## Step 1 — Choose auditor (you)

- [ ] Select an **independent** firm or researcher (not the primary implementer)
- [ ] Confirm scope matches `docs/audits/2026-09-01-security-audit-scope.md`
- [ ] Agree timeline, deliverable format, and remediation process

Suggested outreach email opening:

> We are preparing Sudharma Network for a future mainnet launch (Proof-of-Work, GPU-only mining, 51M hard cap). Engineering is merged on `main`; we need an independent security review before genesis freeze. Scope document attached in repo: `docs/audits/2026-09-01-security-audit-scope.md`.

---

## Step 2 — Send auditor package

Share these with the auditor (public repo links are fine):

| Document | Purpose |
| --- | --- |
| `docs/audits/2026-09-01-security-audit-scope.md` | Full review scope + file map |
| `docs/audits/2026-09-01-security-audit-brief.md` | Executive brief for auditors |
| `docs/audits/2026-08-31-mainnet-readiness.md` | What engineering freeze includes |
| `docs/audits/2026-08-31-mainnet-gpu-mining-architecture.md` | GPU mining threat model |
| `docs/audits/2026-08-31-pool-mining-architecture.md` | Pool / Stratum semantics |

Auditor verification commands:

```bash
git clone https://github.com/sudharma-networks/sudharma.git
cd sudharma
git checkout a63fe384a75a8857f02a78b612be5b9b0a233cb8
bash ./scripts/verify-mainnet-merge-readiness.sh pr77
bash ./scripts/mainnet-monetary-rehearsal.sh
go test ./... -count=1
```

---

## Step 3 — Record kickoff in private vault (do not commit filled form)

Copy `docs/audits/2026-09-01-security-audit-kickoff-record.template.json` to your private evidence folder and fill:

- `auditor_name` / `auditor_contact`
- `kickoff_date`
- `commit_audited`
- `status`: `commissioned` → `in_progress` → `report_received`

---

## Step 4 — During audit

- [ ] Provide auditor read-only access to staging if needed (no production keys)
- [ ] Track findings in private vault (severity + remediation owner)
- [ ] Do **not** open genesis freeze or activation PRs until critical/high items are resolved or explicitly accepted in writing

---

## Step 5 — Close the gate (after signed report)

1. Record completed audit in private vault using `docs/audits/2026-08-31-security-audit-evidence-template.md`
2. Remediate critical/high findings on `main` (separate PRs)
3. Re-run verification on remediated commit
4. Owner written sign-off that audit gate is satisfied
5. **Then** proceed to genesis timestamp freeze — not before

Note: `params/readiness.go` gate `independent-security-audit` is updated only in a dedicated PR after human review of the signed report — not before.

---

## Explicit out of scope for auditors (unless separately contracted)

- Mainnet seed production deploy
- AWS IAM / live infrastructure pentest (optional add-on)
- Economic/market analysis
- Legal / regulatory opinion

---

## Related

- Evidence after audit: `docs/audits/2026-08-31-security-audit-evidence-template.md`
- Genesis freeze (after audit): `docs/audits/2026-08-31-mainnet-genesis-freeze-template.md`
- Launch decision: `docs/audits/2026-08-31-owner-signoff-templates.md`
