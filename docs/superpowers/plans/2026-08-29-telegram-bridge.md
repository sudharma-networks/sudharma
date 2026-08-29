# Sudharma Telegram Bridge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a fail-closed GitHub IssueOps bridge that can dry-run or publish one validated plain-text announcement to the official Sudharma Telegram channel without exposing Telegram credentials.

**Architecture:** A small dependency-free Node.js module validates issue payloads, trusted author associations, marker syntax, Telegram text length, and idempotency markers. A GitHub Actions workflow runs only on explicit Telegram labels, uses per-issue concurrency, performs dry-runs without Telegram credentials, and publishes with the Bot API only after all gates pass. GitHub issue comments form the audit trail.

**Tech Stack:** Node.js built-ins (`node:test`, `assert`, `fs`), GitHub Actions, GitHub CLI available on hosted runners, Telegram Bot API over HTTPS/curl.

**Spec:** `docs/superpowers/specs/2026-08-29-telegram-bridge-design.md`

## Global Constraints

- Phase 1 is official-channel plain-text publishing only.
- Never use a personal Telegram login, phone number, OTP, session, or session database.
- `TELEGRAM_BOT_TOKEN` must exist only as a GitHub Actions secret and must never be committed or printed.
- The destination must come from `TELEGRAM_CHANNEL_ID`, never from issue content.
- Publishing requires the exact `telegram:publish-approved` label and trusted author association `OWNER`, `MEMBER`, or `COLLABORATOR`.
- Dry-run uses `telegram:dry-run` and must never call Telegram.
- Telegram message text must be non-empty and at most 4096 Unicode code points.
- Duplicate marker pairs are rejected.
- Per-issue idempotency must prevent duplicate sends after reruns or label reapplication.
- Workflow permissions must be limited to `contents: read` and `issues: write`.
- No AWS, mainnet, consensus, wallet-key, miner, Seed-1, or Seed-2 code may be changed.

---

### Task 1: Deterministic Telegram command validation

**Files:**
- Create: `scripts/telegram/bridge-core.test.js`
- Create: `scripts/telegram/bridge-core.js`

**Interfaces:**
- Consumes: GitHub issue body text, GitHub `author_association`, prior issue comment bodies, issue number.
- Produces: `parseTelegramMessage(body) -> string`, `isTrustedAssociation(value) -> boolean`, `publishedMarker(issueNumber) -> string`, `hasPublishedMarker(commentBodies, issueNumber) -> boolean`.

- [ ] **Step 1: Write failing parser/security tests**

Create `scripts/telegram/bridge-core.test.js` using `node:test` and `node:assert/strict`. Cover exact text preservation, missing/duplicate markers, empty content, >4096 code points, trusted associations, and exact idempotency markers.

- [ ] **Step 2: Run RED verification**

Run:

```bash
node --test scripts/telegram/bridge-core.test.js
```

Expected: FAIL because `./bridge-core.js` does not exist yet.

- [ ] **Step 3: Implement minimal pure functions**

Create `scripts/telegram/bridge-core.js` with these constants and behaviors:

```js
const CONTROL = '<!-- sudharma-telegram-bridge:v1 -->';
const BEGIN = 'TELEGRAM_MESSAGE_BEGIN';
const END = 'TELEGRAM_MESSAGE_END';
const TRUSTED = new Set(['OWNER', 'MEMBER', 'COLLABORATOR']);
const MAX_CODE_POINTS = 4096;
```

`parseTelegramMessage` must require exactly one control marker, exactly one begin marker, exactly one end marker, require marker order `CONTROL < BEGIN < END`, trim only the wrapper boundary newline/whitespace around the message, preserve internal whitespace/newlines, reject an empty message, and reject message length above 4096 Unicode code points using `[...message].length`.

`publishedMarker(issueNumber)` returns exactly:

```text
<!-- sudharma-telegram-published:v1 issue=NUMBER -->
```

`hasPublishedMarker` performs an exact substring check for that issue-specific marker across supplied comment bodies.

- [ ] **Step 4: Run GREEN verification**

Run:

```bash
node --test scripts/telegram/bridge-core.test.js
```

Expected: all tests PASS.

- [ ] **Step 5: Commit Task 1**

Commit tests first for the RED checkpoint, then implementation for GREEN.

---

### Task 2: Safe CLI validator and GitHub IssueOps workflow

**Files:**
- Create: `scripts/telegram/validate-event.test.js`
- Create: `scripts/telegram/validate-event.js`
- Create: `.github/workflows/telegram-publish.yml`

**Interfaces:**
- Consumes: `GITHUB_EVENT_PATH`, `TELEGRAM_MODE` (`dry-run` or `publish`).
- Produces: a UTF-8 message file at the path provided by `TELEGRAM_MESSAGE_FILE`; exits non-zero on malformed or unauthorized input.

- [ ] **Step 1: Write failing validator tests**

Tests construct temporary GitHub issue-event JSON files and verify that trusted labeled events produce the exact message file while untrusted associations, wrong labels, malformed bodies, and pull-request-shaped issue payloads are rejected.

- [ ] **Step 2: Run RED verification**

Run:

```bash
node --test scripts/telegram/validate-event.test.js
```

Expected: FAIL because `validate-event.js` does not exist yet.

- [ ] **Step 3: Implement the validator CLI**

`validate-event.js` must read the event JSON, require a normal issue event (not an issue representing a pull request), require `OWNER|MEMBER|COLLABORATOR`, require the event label to match mode exactly (`telegram:dry-run` or `telegram:publish-approved`), parse text through `bridge-core.js`, and write only the validated message to `TELEGRAM_MESSAGE_FILE`.

It must never read or print `TELEGRAM_BOT_TOKEN`.

- [ ] **Step 4: Run validator GREEN verification**

Run both Telegram test files with:

```bash
node --test scripts/telegram/*.test.js
```

Expected: all tests PASS.

- [ ] **Step 5: Add the workflow**

Create `.github/workflows/telegram-publish.yml` triggered only by `issues: [labeled]`, with:

```yaml
permissions:
  contents: read
  issues: write

concurrency:
  group: telegram-issue-${{ github.event.issue.number }}
  cancel-in-progress: false
```

The job condition must allow only the two Telegram labels and trusted author associations. The workflow validates the event before any publish step.

Dry-run path: validate, report the Unicode character count to the issue, and never reference `TELEGRAM_BOT_TOKEN` or call `api.telegram.org`.

Publish path: verify `TELEGRAM_CHANNEL_ID` and `TELEGRAM_BOT_TOKEN` are present; query issue comments for the issue-specific published marker; skip if already present; POST JSON to Telegram `sendMessage` with `chat_id`, `text`, and `disable_web_page_preview: true`; parse `ok === true` and integer `result.message_id`; then post an audit comment containing the published marker and Telegram message ID.

Use `set +x`, do not print the Bot API URL, and do not echo secret environment variables.

- [ ] **Step 6: Add workflow static-safety tests**

Extend `validate-event.test.js` to read `.github/workflows/telegram-publish.yml` and assert that it contains least-privilege permissions, per-issue concurrency, both explicit labels, no `pull_request_target`, no AWS references, and that dry-run and publish are separate conditional steps.

- [ ] **Step 7: Run all Telegram tests**

Run:

```bash
node --test scripts/telegram/*.test.js
```

Expected: all tests PASS.

---

### Task 3: CI integration and Android operator guide

**Files:**
- Modify: `.github/workflows/ci.yml`
- Create: `docs/telegram-bridge.md`

**Interfaces:**
- Consumes: existing repository CI and GitHub/Telegram Android UI.
- Produces: automatic Telegram bridge test coverage on PRs/pushes and an owner-safe one-time setup/runbook.

- [ ] **Step 1: Add Telegram tests to CI**

Add a Node.js setup step using Node 22 and run:

```bash
node --test scripts/telegram/*.test.js
```

Do not add third-party npm dependencies.

- [ ] **Step 2: Write Android-only operator guide**

Document:

1. Create a bot in `@BotFather` using `/newbot`.
2. Add the bot to the official Sudharma announcement channel as admin with only posting capability required for Phase 1.
3. In GitHub mobile browser, add Actions secret `TELEGRAM_BOT_TOKEN`.
4. Add Actions variable `TELEGRAM_CHANNEL_ID` with the official channel username/ID; never put the token in a variable.
5. Never paste the token into ChatGPT, issues, screenshots, commits, or Telegram messages.
6. Create an issue using the exact v1 marker block.
7. Apply `telegram:dry-run` first and confirm the audit comment.
8. Apply `telegram:publish-approved` only after dry-run passes.
9. Confirm one Telegram post and one GitHub success marker.
10. Reapply the label/rerun and confirm no duplicate Telegram message is produced.
11. Token rotation: revoke/regenerate in BotFather and replace the GitHub secret.
12. Recovery: remove bot admin rights to immediately stop Telegram publishing.

- [ ] **Step 3: Run complete local verification**

Run:

```bash
go test ./... -count=1
node --test scripts/telegram/*.test.js
```

Expected: all Go and Telegram tests PASS.

- [ ] **Step 4: Open a draft PR**

Open a draft PR from `feature/telegram-bridge` to `main`. Keep it unmerged until CI is green and the owner has completed the one-time BotFather/GitHub secret setup required for a controlled live smoke test.

- [ ] **Step 5: Live smoke test after owner setup**

Use one explicit test announcement, confirm exactly one Telegram channel message and the returned Telegram message ID in the GitHub audit comment, then retry the same issue and verify idempotency prevents a duplicate.
