# Security audit outreach — tight budget (owner: Kk)

**Owner email:** vkw143download@gmail.com  
**Budget:** Very tight (student / open-source project, India)  
**Honest limit:** AI self-scan tools and internal review **do not** close the `independent-security-audit` mainnet gate. You still need a **signed report from an independent third party** (firm, university lab with MOU, or qualified researcher under contract).

This doc prioritizes **lowest cost first**, then **phased paid scope** if any budget appears later.

---

## Tier 0 — Free (do this week)

| Action | Cost | Closes audit gate? |
| --- | --- | --- |
| [Trail of Bits office hours](https://trailofbits.com/services/) | Free | No — scoping only |
| Run Sudharma verification scripts on `main` | Free | No — engineering baseline |
| University professor / security lab outreach | Free if MOU | **Yes**, if lab delivers signed report |
| GitHub public **Security** tab → private vulnerability report | Free | No — ongoing, not pre-mainnet gate |

**Engineering baseline (you or agent can run anytime):**

```bash
bash ./scripts/verify-mainnet-merge-readiness.sh pr77
bash ./scripts/check-tracked-secrets_test.sh
go test ./... -count=1
```

---

## Tier 1 — Ask for student / OSS discount (email below)

Full L1 protocol audits often cost **USD 30k–150k+**. With tight budget, ask for:

1. **Phased scope** — Phase A only: monetary cap + `MintSupplyFor` + genesis isolation (~smallest critical surface)
2. **Rolling commit review** (Sigma Prime) — incremental PR review vs full audit
3. **Public contest + private review hybrid** (Sherlock) — sometimes cheaper than solo firm
4. **Pro bono / academic** — university capstone with faculty sign-off

---

## Email 1 — Trail of Bits (FREE office hours first)

**Send from:** vkw143download@gmail.com  
**To:** book via https://trailofbits.com/services/ (office hours form)  
**Or To:** info@trailofbits.com  
**Subject:** Free office hours — pre-mainnet Go PoW L1 scoping (tight budget)

```
Hello,

I am Kk, owner of Sudharma Network — an open-source Go Proof-of-Work L1 (student-built, India). Mainnet is NOT live yet.

We need guidance on scoping an independent pre-mainnet review on a very tight budget.

Repo: https://github.com/sudharma-networks/sudharma
Scope doc: https://github.com/sudharma-networks/sudharma/blob/main/docs/audits/2026-09-01-security-audit-scope.md

Could we book a free technical office hours session to discuss:
- Minimum viable audit scope for a Go L1 (not EVM-only)
- Whether phased review (monetary policy + consensus first) is realistic for us
- Any student / open-source programs you recommend

Contact: vkw143download@gmail.com

Thank you,
Kk
Sudharma Network
```

---

## Email 2 — Sigma Prime (protocol firm — ask phased pricing)

**Send from:** vkw143download@gmail.com  
**To:** use form at https://sigmaprime.io/services/blockchain-protocol-audit/  
**Subject:** Phased protocol audit quote — Go PoW L1, student project, tight budget

```
Hello,

I am Kk, project owner of Sudharma Network (Go, Proof-of-Work L1, 51M hard cap). Mainnet engineering is on GitHub main; launch is gated and not authorized.

We are a student-built open-source project in India with a very tight budget. We cannot afford a full L1 audit upfront.

Would you quote a **Phase A** review limited to:
- Monetary policy / 51M cap enforcement (params/, consensus/, blockchain/state.go)
- Network identity + genesis isolation (mainnet vs testnet)
- ProcessBlockFor / reorg safety on subsidy minting

Repo: https://github.com/sudharma-networks/sudharma
Brief: https://github.com/sudharma-networks/sudharma/blob/main/docs/audits/2026-09-01-security-audit-brief.md

Also interested in **rolling commit review** pricing if that fits better.

Contact: vkw143download@gmail.com

Thank you,
Kk
```

---

## Email 3 — University / academic lab (often lowest path to signed report)

**Send from:** vkw143download@gmail.com  
**To:** cybersecurity / blockchain faculty at your university (or nearby IIT / NIT CS dept)  
**Subject:** Capstone / research collaboration — pre-mainnet security review of open-source PoW L1

```
Dear Professor [NAME],

I am Kk, a student developer on Sudharma Network — an open-source Proof-of-Work blockchain built in India (Go, public GitHub).

We have merged mainnet-readiness engineering but cannot launch until an independent security review is complete. Commercial audit quotes are beyond our budget.

Would your lab or a graduate capstone team be interested in a **structured security review** under faculty supervision? We can provide:
- Full public source: https://github.com/sudharma-networks/sudharma
- Written scope: docs/audits/2026-09-01-security-audit-scope.md
- Test harness and verification scripts

Deliverable we need: a signed review report (findings + severity + remediation status) suitable for a pre-mainnet launch checklist.

Happy to present the architecture or align scope with your course requirements.

Contact: vkw143download@gmail.com

Thank you,
Kk
Sudharma Network
```

---

## What does NOT count as the audit gate

- ChatGPT / Claude “audit my code”
- MimoAudit, Krait, Slither (EVM), generic AI scanners
- Internal team review only
- Public testnet time without written third-party report

Use those for **preparation**, not for closing the gate.

---

## Realistic budget planning

| Scope | Rough market range | Suggestion for Sudharma |
| --- | --- | --- |
| Full L1 protocol audit | USD 50k–150k+ | Not now — ask phased |
| Phase A (monetary + consensus only) | Ask firms | **Start here** |
| Rolling PR review (months) | Lower than full audit | Good if you keep shipping |
| University supervised review | USD 0 (time trade) | **Best fit for tight budget** |
| Trail of Bits office hours | USD 0 | Do first |

---

## After you send any email

Save privately (do not commit secrets):

`docs/audits/2026-09-01-security-audit-kickoff-record.template.json`

Set `"owner_signoff": "Kk"`, `"auditor.contact": "vkw143download@gmail.com"`, `"status": "commissioned"`.

---

## Gmail steps (2 minutes)

1. Open https://mail.google.com (log in as vkw143download@gmail.com)
2. **Compose**
3. Paste **Email 1** (Trail of Bits) first — it's free
4. Send
5. Same for **Email 3** to a professor you know
6. Send **Email 2** when ready for pricing
