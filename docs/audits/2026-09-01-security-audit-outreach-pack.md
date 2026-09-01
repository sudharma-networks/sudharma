# Security audit outreach pack — for owner (Kk)

**Important:** Sudharma cannot close the `independent-security-audit` gate without a **signed report from an independent third party**. An automated agent cannot hire firms, send email from your account, or substitute for that review.

This pack gives you **copy-paste emails** and **contact links** so you can reach auditors in about 10 minutes from your own email.

**Tight budget?** Use `docs/audits/2026-09-01-security-audit-budget-outreach-kk.md` (phased quotes, university path, free office hours).

---

## Best-fit firms for Sudharma (Go PoW L1, not EVM-only)

| Firm | Why | How to contact |
| --- | --- | --- |
| **Sigma Prime** | Protocol / consensus audits; Go, Rust; L1 experience | [Request scoping call](https://sigmaprime.io/services/blockchain-protocol-audit/) (form on page) |
| **Hashlock** | L1 chain audits; PoW/PoS consensus called out | [hashlock.com](https://hashlock.com/services/l1-chain-audit) → “Book consultation” / contact form |
| **Trail of Bits** | Blockchain full-stack; Go; Bitcoin/ETH history | [trailofbits.com/contact](https://trailofbits.com/contact) or **info@trailofbits.com** |
| **Runtime Verification** | Formal methods / high-assurance consensus (premium, long lead) | [runtimeverification.com](https://runtimeverification.com/) contact |
| **Sherlock** | Researcher network; good if budget-sensitive contest + review hybrid | [sherlock.xyz](https://sherlock.xyz/) contact |

**Free first step (Trail of Bits):** [Book technical office hours](https://trailofbits.com/services/) — 1 hour, no sales pitch, to ask if scope fits before paying.

---

## Email template — send from **your** mailbox

**To:** (pick one firm above)  
**Subject:** Pre-mainnet security review — Sudharma Network (Go PoW L1, 51M cap)

```
Hello,

I am Kk, project owner of Sudharma Network (open-source Proof-of-Work L1, native coin SUDH).

We have merged mainnet engineering to GitHub main but have NOT launched mainnet. We need an independent security review before genesis freeze and launch.

Repository: https://github.com/sudharma-networks/sudharma
Target commit: 39301b097285bcf5dc942bd4ce98abdb96ed843e (or latest main)

Scope summary:
- Monetary policy: 51,000,000 SUDH hard cap, 40-epoch emission (Go)
- Consensus / block processing / reorg (Go)
- P2P sync and network identity isolation (testnet vs mainnet)
- Public HTTP RPC + GPU mining API (testnet live)
- Wallet key custody boundaries (CLI + Android)
- Pool / Stratum reference stack (PPS/PPLNS/SOLO/FPPS)
- Operator deployment safety (manual-only CI gates)

Full scope document (public):
https://github.com/sudharma-networks/sudharma/blob/main/docs/audits/2026-09-01-security-audit-scope.md

Executive brief:
https://github.com/sudharma-networks/sudharma/blob/main/docs/audits/2026-09-01-security-audit-brief.md

We are a student-built project in India; please share ballpark pricing and timeline for a pre-mainnet protocol review (not smart-contract-only).

Thank you,
Kk
Project owner, Sudharma Network
https://github.com/sudharma-networks/sudharma
```

Attach nothing sensitive. All links are public.

---

## After you send (private vault)

Update your copy of `docs/audits/2026-09-01-security-audit-kickoff-record.template.json`:

- `status`: `"commissioned"`
- `auditor.firm_or_researcher`: firm name
- `kickoff_date`: today’s ISO8601 UTC
- `repository_commit_audited`: latest `main` SHA

---

## If budget is zero today

| Option | Notes |
| --- | --- |
| Trail of Bits **office hours** | Free scoping call; not a full audit |
| University security lab | Partner with a CS/security department (MOU + advisor) |
| Public **bug bounty** (later) | After testnet hardening; not a substitute for pre-mainnet gate |
| **Rolling commit review** (Sigma Prime) | Cheaper incremental PR reviews vs full audit |

Do **not** mark the audit gate complete without a signed independent report.

---

## What Sudharma engineering can do while you wait

```bash
bash ./scripts/verify-mainnet-merge-readiness.sh pr77
bash ./scripts/check-tracked-secrets_test.sh
go test ./... -count=1
```

Owner checklist: `docs/audits/2026-09-01-security-audit-kickoff-checklist.md`
