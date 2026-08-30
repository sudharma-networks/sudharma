# Telegram Community Bot Phase 2A Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a commands-only Telegram community bot for `@sudharma_community` that answers fixed help commands and converts validated `/report` messages into deduplicated public GitHub issues.

**Architecture:** Extend the existing Telegram tooling beside, not inside, the Phase 1 announcement path. Pure parsing/formatting/rate-limit logic lives in `community-core.js`; one dependency-injectable `community-worker.js` performs Telegram/GitHub I/O and acknowledges only the contiguous successfully handled update prefix. Initial rollout is manual-only through `workflow_dispatch`; a five-minute schedule is enabled in a separate activation change only after bootstrap and live smoke tests succeed.

**Tech Stack:** Node.js 22 built-ins (`node:test`, `fetch`), Telegram Bot API, GitHub REST API, GitHub Actions YAML.

**Spec:** `docs/superpowers/specs/2026-08-30-telegram-community-phase2a-design.md`

## Global Constraints

- Official community destination is fixed to Telegram username `sudharma_community`; incoming private chats and other groups are ignored.
- Group Privacy Mode remains enabled and the bot must be a normal group member, not an administrator.
- Supported commands are `/help`, `/rules`, `/testnet`, `/miner`, and `/report`; ordinary group conversation is ignored.
- `/report` text is 20–2000 Unicode code points and must be mirrored only after GitHub `@mention` neutralization.
- Never copy Telegram numeric user IDs into GitHub issue content.
- Maximum 3 new reports per poll run, 10 Telegram-created reports per rolling hour, and 20 bot replies per poll run.
- Existing `TELEGRAM_BOT_TOKEN` in GitHub environment `telegram-publishing` is reused; no new secret is introduced.
- Workflow permissions remain `contents: read` and `issues: write`; never add `contents: write`.
- Bootstrap checks `getWebhookInfo` and fails closed if a webhook exists before using `deleteWebhook(drop_pending_updates=true)`.
- Initial rollout has no recurring schedule. Enable the five-minute schedule only after controlled `/help` and `/report` smoke tests and deduplication review.
- Phase 1 announcement workflow must remain operational and unchanged by Phase 2A implementation.
- No mainnet activation, unrestricted GPU mining, GPU-PoW deployment to Seed-1/Seed-2, monetary-rule changes, or wallet/node credential access.

---

### Task 1: Pure Community Command and Report Contract

**Files:**
- Create: `scripts/telegram/community-core.test.js`
- Create: `scripts/telegram/community-core.js`

**Interfaces:**
- Produces: `OFFICIAL_GROUP_USERNAME`, `REPORT_MIN_CODE_POINTS`, `REPORT_MAX_CODE_POINTS`, `isOfficialChat(message)`, `parseCommunityUpdate(update, botUsername)`, `neutralizeGithubMentions(text)`, `telegramMessageLink(message)`, `reportMarker(updateId)`, `hasReportMarker(issueBody, updateId)`, `buildReportIssue({updateId, reportText, message, now})`, `staticReply(kind)`, `canCreateReport({createdThisRun, createdLastHour})`.
- `parseCommunityUpdate` returns one of `{kind:'ignore'}`, `{kind:'welcome', message}`, or `{kind:'command', command, args, message}`.

- [ ] **Step 1: Write failing tests for official-chat and command parsing**

```js
const test = require('node:test');
const assert = require('node:assert/strict');
const {
  isOfficialChat,
  parseCommunityUpdate,
} = require('./community-core.js');

const groupMessage = (text, extra = {}) => ({
  message_id: 44,
  date: 1788048000,
  chat: { id: -100123, type: 'supergroup', username: 'sudharma_community' },
  text,
  ...extra,
});

test('accepts only the official public group', () => {
  assert.equal(isOfficialChat(groupMessage('/help')), true);
  assert.equal(isOfficialChat({ ...groupMessage('/help'), chat: { id: -1, type: 'supergroup', username: 'other' } }), false);
  assert.equal(isOfficialChat({ ...groupMessage('/help'), chat: { id: 1, type: 'private', username: 'sudharma_community' } }), false);
});

test('ordinary conversation is ignored', () => {
  assert.deepEqual(parseCommunityUpdate({ update_id: 1, message: groupMessage('hello everyone') }, 'SudharmaNetworkBot'), { kind: 'ignore' });
});

test('bot-addressed commands parse and commands for another bot are ignored', () => {
  assert.equal(parseCommunityUpdate({ update_id: 2, message: groupMessage('/help@SudharmaNetworkBot') }, 'SudharmaNetworkBot').command, 'help');
  assert.equal(parseCommunityUpdate({ update_id: 3, message: groupMessage('/help@OtherBot') }, 'SudharmaNetworkBot').kind, 'ignore');
});
```

- [ ] **Step 2: Run the test and verify RED**

Run: `node --test scripts/telegram/community-core.test.js`
Expected: FAIL because `community-core.js` does not exist.

- [ ] **Step 3: Add failing tests for report safety, link construction, unknown commands, and welcome events**

```js
test('unknown explicit command maps to help', () => {
  const result = parseCommunityUpdate({ update_id: 4, message: groupMessage('/wat') }, 'SudharmaNetworkBot');
  assert.equal(result.kind, 'command');
  assert.equal(result.command, 'help');
});

test('new members create a generic welcome action without identities', () => {
  const message = groupMessage('', { new_chat_members: [{ id: 999, first_name: 'Private Name' }] });
  const result = parseCommunityUpdate({ update_id: 5, message }, 'SudharmaNetworkBot');
  assert.equal(result.kind, 'welcome');
  assert.doesNotMatch(result.text, /999|Private Name/);
});
```

Add report tests proving 19 code points rejects, 20 accepts, 2000 accepts, 2001 rejects; `@alice` becomes `@\u200balice`; report bodies contain `<!-- sudharma-telegram-community-report:v1 update_id=123 -->`, contain `https://t.me/sudharma_community/44`, and never contain `from.id`.

- [ ] **Step 4: Implement the minimal pure core**

Implement constants and pure functions only. Parse command names case-insensitively from the first token, accept optional `@BotUsername` case-insensitively, use `[...text].length` for Unicode limits, and build plain-text replies with no Telegram markup assumptions.

`telegramMessageLink(message)` must return exactly `https://t.me/sudharma_community/${message.message_id}` only for an official-group message.

`canCreateReport` must return `{allowed:false, reason:'run-limit'}` at `createdThisRun >= 3`, `{allowed:false, reason:'hour-limit'}` at `createdLastHour >= 10`, else `{allowed:true}`.

- [ ] **Step 5: Run focused tests and full Telegram tests**

Run: `node --test scripts/telegram/community-core.test.js`
Expected: PASS.

Run: `node --test scripts/telegram/*.test.js`
Expected: all Telegram tests PASS.

- [ ] **Step 6: Commit**

```bash
git add scripts/telegram/community-core.js scripts/telegram/community-core.test.js
git commit -m "feat(telegram): add community command core"
```

---

### Task 2: Poll Worker With Contiguous Update Acknowledgment

**Files:**
- Create: `scripts/telegram/community-worker.test.js`
- Create: `scripts/telegram/community-worker.js`

**Interfaces:**
- Consumes Task 1 exports.
- Produces: `createWorker({telegram, github, now, logger})`, whose returned object exposes `bootstrap()` and `poll()`.
- Telegram dependency contract: `getMe()`, `getWebhookInfo()`, `deleteWebhook({dropPendingUpdates})`, `getUpdates({offset, limit, timeout})`, `sendMessage(payload)`.
- GitHub dependency contract: `listRecentIssues({since})`, `createIssue({title, body})`.

- [ ] **Step 1: Write RED tests for ignored updates, help replies, and contiguous acknowledgment**

```js
test('poll acknowledges ignored + successful updates only through contiguous prefix', async () => {
  const calls = [];
  const worker = createWorker({
    telegram: fakeTelegram({
      updates: [
        { update_id: 10, message: otherGroup('/help') },
        { update_id: 11, message: official('/help') },
      ],
      onGetUpdates: (args) => calls.push(args),
    }),
    github: fakeGithub(),
    now: () => new Date('2026-08-30T02:00:00Z'),
    logger: silentLogger,
  });
  await worker.poll();
  assert.deepEqual(calls.at(-1), { offset: 12, limit: 1, timeout: 0 });
});
```

Add a failure test where update 11 reply throws and update 12 exists; assert the final acknowledgment offset is `11` (confirming only update 10), and update 12 is never processed.

- [ ] **Step 2: Run RED**

Run: `node --test scripts/telegram/community-worker.test.js`
Expected: FAIL because worker does not exist.

- [ ] **Step 3: Implement Telegram and GitHub adapters plus dependency-injectable worker**

Use Node 22 global `fetch` in production adapters. Build Telegram URLs from the secret at runtime without logging the token. Parse every API response and require Telegram `ok === true`; require GitHub `2xx` for issue/list operations.

`poll()` algorithm:

```js
const bot = await telegram.getMe();
const updates = await telegram.getUpdates({ limit: 100, timeout: 0 });
let lastConfirmed = null;
for (const update of updates.sort((a, b) => a.update_id - b.update_id)) {
  try {
    await handleUpdate(update, bot.username);
    lastConfirmed = update.update_id;
  } catch (error) {
    break;
  }
}
if (lastConfirmed !== null) {
  await telegram.getUpdates({ offset: lastConfirmed + 1, limit: 1, timeout: 0 });
}
```

Never continue past a failed actionable update.

- [ ] **Step 4: Implement static-command/welcome handling and 20-reply cap**

For `help`, `rules`, `testnet`, `miner`, invalid report guidance, throttle responses, and welcome responses, call `sendMessage` with plain `text`, `chat_id` from the verified official message, `reply_parameters: {message_id}`, optional `message_thread_id`, and `link_preview_options: {is_disabled:true}`. Do not set `parse_mode`.

Once 20 replies were sent in the current run, deliberately ignore further non-report informational actions and allow them to be acknowledged without reply. Report issue creation must still obey its independent report limits.

- [ ] **Step 5: Run worker and full Telegram tests**

Run: `node --test scripts/telegram/community-worker.test.js`
Expected: PASS.

Run: `node --test scripts/telegram/*.test.js`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add scripts/telegram/community-worker.js scripts/telegram/community-worker.test.js
git commit -m "feat(telegram): add community polling worker"
```

---

### Task 3: `/report` GitHub Intake, Rolling Limit, and Retry Deduplication

**Files:**
- Modify: `scripts/telegram/community-worker.test.js`
- Modify: `scripts/telegram/community-worker.js`

**Interfaces:**
- Uses Task 1 `reportMarker`, `hasReportMarker`, `buildReportIssue`, `canCreateReport`.
- `github.listRecentIssues({since})` returns normalized `{html_url, body, created_at}` objects across enough pages to reach `since` or exhaust results.

- [ ] **Step 1: Add RED test for one report -> one issue -> URL reply**

```js
test('/report creates one structured GitHub issue and replies with its URL', async () => {
  const github = fakeGithub({ createdUrl: 'https://github.com/sudharma-networks/sudharma/issues/999' });
  const telegram = fakeTelegram({ updates: [{ update_id: 77, message: official('/report miner exits after startup on RX 580 with OpenCL error') }] });
  await createWorker({ telegram, github, now: fixedNow, logger: silentLogger }).poll();
  assert.equal(github.created.length, 1);
  assert.match(github.created[0].body, /update_id=77/);
  assert.match(telegram.sent[0].text, /issues\/999/);
});
```

- [ ] **Step 2: Add RED retry-dedup and rate-limit tests**

Provide a recent issue whose body contains the same update marker; assert no new issue is created and the existing issue URL is replied. Provide 10 recent Phase 2A issues inside one hour; assert no new issue and throttle response. Provide three valid reports in one batch followed by a fourth; assert only three new issues.

- [ ] **Step 3: Run RED**

Run: `node --test scripts/telegram/community-worker.test.js`
Expected: FAIL on report workflow assertions.

- [ ] **Step 4: Implement report processing**

Load recent issues once per poll using `since = new Date(now().getTime() - 60*60*1000).toISOString()`. Before creating any issue, scan normalized issue bodies for the exact update marker. If found, reuse its URL. Otherwise apply `canCreateReport` and create the issue only when allowed.

GitHub production adapter must paginate `/repos/${GITHUB_REPOSITORY}/issues?state=all&since=<ISO>&per_page=100&sort=created&direction=desc`, stopping when a page is empty, shorter than 100, or its oldest `created_at` is earlier than `since`. Pull-request-shaped entries may be ignored for counting/dedup.

- [ ] **Step 5: Run tests**

Run: `node --test scripts/telegram/community-worker.test.js`
Expected: PASS.

Run: `node --test scripts/telegram/*.test.js`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add scripts/telegram/community-worker.js scripts/telegram/community-worker.test.js
git commit -m "feat(telegram): route community reports to GitHub"
```

---

### Task 4: Fail-Closed Bootstrap Mode

**Files:**
- Modify: `scripts/telegram/community-worker.test.js`
- Modify: `scripts/telegram/community-worker.js`

**Interfaces:**
- `bootstrap()` verifies bot identity, requires `getWebhookInfo().url` to be empty, then calls `deleteWebhook({dropPendingUpdates:true})` and exits without `sendMessage`, `createIssue`, or `getUpdates` polling.

- [ ] **Step 1: Add RED bootstrap tests**

```js
test('bootstrap fails closed when a webhook is already configured', async () => {
  const telegram = fakeTelegram({ webhookUrl: 'https://existing.example/hook' });
  await assert.rejects(() => createWorker({ telegram, github: fakeGithub(), now: fixedNow, logger: silentLogger }).bootstrap(), /webhook/i);
  assert.equal(telegram.deletedWebhook.length, 0);
  assert.equal(telegram.sent.length, 0);
});

test('bootstrap drops pending updates only when webhook is empty', async () => {
  const telegram = fakeTelegram({ webhookUrl: '' });
  await createWorker({ telegram, github: fakeGithub(), now: fixedNow, logger: silentLogger }).bootstrap();
  assert.deepEqual(telegram.deletedWebhook, [{ dropPendingUpdates: true }]);
  assert.equal(telegram.sent.length, 0);
});
```

- [ ] **Step 2: Run RED**

Run: `node --test scripts/telegram/community-worker.test.js`
Expected: FAIL on bootstrap behavior.

- [ ] **Step 3: Implement bootstrap and CLI entry point**

At the bottom of `community-worker.js`, guard production execution with `if (require.main === module)`. Require `TELEGRAM_BOT_TOKEN`, `GITHUB_TOKEN`, `GITHUB_REPOSITORY`, and `COMMUNITY_MODE`. Accept only `bootstrap` or `poll`; unknown mode exits non-zero. Construct adapters without ever printing credential values.

- [ ] **Step 4: Run tests and static secret scan**

Run: `node --test scripts/telegram/*.test.js`
Expected: PASS.

Run: `bash ./scripts/check-tracked-secrets_test.sh`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add scripts/telegram/community-worker.js scripts/telegram/community-worker.test.js
git commit -m "feat(telegram): add safe community bootstrap"
```

---

### Task 5: Manual-Only GitHub Actions Rollout Workflow

**Files:**
- Create: `.github/workflows/telegram-community.yml`
- Create: `scripts/telegram/community-workflow.test.js`

**Interfaces:**
- Workflow input `mode` is an explicit choice of `bootstrap` or `poll`.
- Uses environment `telegram-publishing` and passes existing `TELEGRAM_BOT_TOKEN`, `github.token`, `github.repository`, and selected mode to `community-worker.js`.

- [ ] **Step 1: Write RED workflow contract tests**

Read the workflow as text and assert:

```js
assert.match(workflow, /workflow_dispatch:/);
assert.doesNotMatch(workflow, /schedule:/);
assert.match(workflow, /contents:\s*read/);
assert.match(workflow, /issues:\s*write/);
assert.doesNotMatch(workflow, /contents:\s*write/);
assert.match(workflow, /environment:\s*telegram-publishing/);
assert.match(workflow, /TELEGRAM_BOT_TOKEN:\s*\$\{\{ secrets\.TELEGRAM_BOT_TOKEN \}\}/);
assert.match(workflow, /group:\s*telegram-community-poller/);
assert.match(workflow, /cancel-in-progress:\s*false/);
assert.doesNotMatch(workflow, /aws-actions|amazonaws|AWS_/i);
```

Also read `.github/workflows/telegram-publish.yml` and assert it still contains the Phase 1 exact labels and `api.telegram.org` publish path; do not edit that file in this task.

- [ ] **Step 2: Run RED**

Run: `node --test scripts/telegram/community-workflow.test.js`
Expected: FAIL because workflow does not exist.

- [ ] **Step 3: Create manual-only workflow**

Use `workflow_dispatch.inputs.mode` with required `choice` options `bootstrap` and `poll`. Set `permissions: contents: read, issues: write`, `concurrency.group: telegram-community-poller`, `cancel-in-progress: false`, `runs-on: ubuntu-latest`, `environment: telegram-publishing`, checkout, setup Node 22, then run `node scripts/telegram/community-worker.js` with the four environment variables.

Do not add `schedule` yet.

- [ ] **Step 4: Run Telegram tests and full repository CI-equivalent commands**

Run: `node --test scripts/telegram/*.test.js`
Run: `go test ./... -count=1`
Run: `bash ./scripts/testnet-rehearsal.sh`
Run: `go test -race ./... -count=1`
Expected: PASS. Container build/smoke remains covered by GitHub CI after push/PR.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/telegram-community.yml scripts/telegram/community-workflow.test.js
git commit -m "ci(telegram): add manual community bot workflow"
```

---

### Task 6: Operator Runbook and Pre-Rollout PR Verification

**Files:**
- Modify: `docs/telegram-bridge.md`
- Create: `docs/telegram-community-phase2a.md`

**Interfaces:**
- Documents Android-only one-time Telegram setup and the exact manual workflow sequence.

- [ ] **Step 1: Add Phase 2A runbook**

Document exactly:

1. Add the existing Sudharma bot to `@sudharma_community` as a **normal member**, not admin.
2. In BotFather, verify Group Privacy Mode is **enabled** for this bot.
3. Do not paste or rotate the bot token unless rotation is separately required.
4. After code merges to `main`, manually dispatch `Telegram Community` with `mode=bootstrap`.
5. Confirm bootstrap succeeds before sending test commands.
6. Manually dispatch `mode=poll` immediately after a controlled `/help` message and verify one bot reply.
7. Send a controlled `/report Phase 2A controlled test report ...` message, dispatch `mode=poll`, verify one GitHub issue and one bot reply containing its URL.
8. Re-run `mode=poll` and verify no duplicate issue.
9. Only after those checks, land the separate schedule-activation change.

Include emergency stop: remove bot from community group or remove/disable scheduled workflow; revoking the token stops both Phase 1 and Phase 2 and is therefore a broader emergency action.

- [ ] **Step 2: Update the main Telegram bridge doc**

Change Phase 1 wording that incorrectly describes trusted association if necessary so it reflects the currently implemented trusted GitHub actor gate, then add a short Phase 2A pointer. Do not change Phase 1 workflow behavior.

- [ ] **Step 3: Run documentation-sensitive tests and secret scan**

Run: `node --test scripts/telegram/*.test.js`
Run: `bash ./scripts/check-tracked-secrets_test.sh`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add docs/telegram-bridge.md docs/telegram-community-phase2a.md
git commit -m "docs(telegram): add community Phase 2A runbook"
```

- [ ] **Step 5: Open a draft PR and verify exact-head CI**

Open `feature/telegram-community-phase2a` -> `main` as a draft PR. Verify changed files are limited to Phase 2A scripts/tests/workflow/docs/spec/plan plus any narrowly necessary CI integration. Wait for the full existing CI suite: tracked-secret test, Telegram Node tests, Go tests, local two-node rehearsal, public-testnet container build/smoke, and race detector.

Do not merge while CI is incomplete or failing.

---

### Task 7: Controlled Live Rollout and Schedule Activation

**Files:**
- Modify after smoke tests only: `.github/workflows/telegram-community.yml`
- Modify test after smoke tests only: `scripts/telegram/community-workflow.test.js`

**Interfaces:**
- Activation adds `schedule: - cron: '*/5 * * * *'` and defaults scheduled runs to `COMMUNITY_MODE=poll`, while preserving manual bootstrap/poll dispatch.

- [ ] **Step 1: Merge manual-only Phase 2A only after exact-head CI is green and owner Telegram prerequisites are complete**

Prerequisites: bot is a non-admin member of `@sudharma_community`; Privacy Mode enabled. Do not enable recurring polling before these are confirmed.

- [ ] **Step 2: Run bootstrap manually and inspect job logs**

Expected: bot identity verified; no existing webhook; `deleteWebhook(drop_pending_updates=true)` succeeds; no Telegram reply and no GitHub issue is created.

- [ ] **Step 3: Run controlled `/help` smoke test**

User sends `/help` in the official community group. Dispatch one manual `poll`. Expected: exactly one plain-text help reply in the same group and no GitHub issue.

- [ ] **Step 4: Run controlled `/report` smoke test and retry check**

User sends a non-sensitive controlled report of at least 20 code points. Dispatch `poll`. Expected: exactly one GitHub issue with Phase 2A marker and source link, and exactly one Telegram reply containing that issue URL. Dispatch `poll` again; expected: no duplicate issue.

- [ ] **Step 5: Write RED activation test**

Change `community-workflow.test.js` to require `schedule:` and exact cron `*/5 * * * *`, while preserving all security assertions.

- [ ] **Step 6: Run activation test to verify RED, then add schedule**

Run: `node --test scripts/telegram/community-workflow.test.js`
Expected before YAML change: FAIL because schedule is absent.

Add:

```yaml
on:
  schedule:
    - cron: '*/5 * * * *'
  workflow_dispatch:
    # existing mode input remains
```

Set `COMMUNITY_MODE` so scheduled events use `poll`, while manual events use the selected input.

- [ ] **Step 7: Run full tests, commit, PR, and exact-head CI**

Run Telegram tests locally/CI and full repository CI. Commit as:

```bash
git commit -m "ci(telegram): enable community polling schedule"
```

Open a small activation PR, verify exact-head CI green, then merge.

- [ ] **Step 8: Observe first scheduled run before declaring Phase 2A complete**

Verify the first scheduled run completes successfully with serialized concurrency and does not affect Phase 1 announcements. If anything behaves unexpectedly, disable schedule/remove bot before troubleshooting.
