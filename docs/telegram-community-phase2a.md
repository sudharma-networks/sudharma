# Sudharma Telegram Community Bot — Phase 2A

Status: manual-only rollout until controlled smoke tests pass

Phase 2A extends the existing Sudharma Telegram bot to the public community group `@sudharma_community`. It is intentionally commands-only: ordinary conversation is ignored and the bot has no moderation powers.

## Safety model

- The bot must be a **normal member** of `@sudharma_community`, not an administrator.
- BotFather **Group Privacy Mode must remain enabled**.
- The worker accepts actionable messages only from the exact Telegram chat username `sudharma_community` and chat type `group` or `supergroup`.
- Supported explicit commands are `/help`, `/rules`, `/testnet`, `/miner`, and `/report`.
- Commands addressed to another bot are ignored.
- Unknown explicit commands addressed to this bot receive the help response only.
- Ordinary group conversation is ignored even if Telegram unexpectedly delivers it.
- The bot cannot ban, mute, delete messages, alter group settings, add administrators, manage invite links, or access personal Telegram sessions.
- Phase 2A never sends community-provided text to the official announcements channel.
- No new Telegram secret is introduced. The existing `TELEGRAM_BOT_TOKEN` remains only in GitHub environment `telegram-publishing`.

## `/report` behavior

Use:

```text
/report describe the reproducible testnet/miner/wallet problem here
```

Rules:

- report text must be 20–2000 Unicode code points;
- accepted report text is copied into the public `sudharma-networks/sudharma` GitHub issue tracker;
- never include seed phrases, private keys, passwords, tokens, credentials, personal information, or other sensitive data;
- Telegram photos, videos, documents, voice notes, and other attachments are **not copied** to GitHub in Phase 2A;
- the created GitHub issue includes a link to the public Telegram source message;
- Telegram numeric user IDs and profile names are not copied into the GitHub issue;
- GitHub `@mention` syntax in user text is neutralized before issue creation;
- retries use the Telegram `update_id` marker to avoid creating a duplicate issue.

Abuse controls:

- maximum 3 new GitHub reports in one poll run;
- maximum 10 Telegram-created report issues in a rolling one-hour window;
- maximum 20 bot replies in one poll run.

If reply capacity is exhausted before an actionable update, the worker stops without acknowledging that update so it can be retried in the next run.

## One-time Android Telegram setup

### 1. Add the existing bot to the community group

1. Open Telegram on Android.
2. Open `@sudharma_community`.
3. Open the group information screen.
4. Choose **Add Members**.
5. Search the exact username of the existing Sudharma Network bot created for Phase 1.
6. Add it as a **normal member**.
7. Do **not** promote it to administrator.

Phase 2A does not require any group-admin permission.

### 2. Verify Group Privacy Mode in BotFather

1. Open the verified `@BotFather` chat.
2. Send `/mybots`.
3. Select the existing Sudharma Network bot.
4. Open **Bot Settings**.
5. Open **Group Privacy**.
6. Confirm privacy mode is **enabled**.

Do not disable Privacy Mode for Phase 2A. Privacy Mode is a defense-in-depth control that limits ordinary group-message delivery to the bot.

Do not display, paste, rotate, or resend the bot token while performing this step.

## GitHub workflow

Workflow: **Telegram Community** (`.github/workflows/telegram-community.yml`)

Initial rollout is manual-only. It supports two `workflow_dispatch` modes:

- `bootstrap`
- `poll`

It uses only:

```text
permissions:
  contents: read
  issues: write
```

Runs are serialized under one concurrency group so overlapping poll workers cannot intentionally process the queue together.

## Bootstrap procedure

Run bootstrap only after the bot is a non-admin member of `@sudharma_community` and Group Privacy Mode is confirmed enabled.

From GitHub on Android:

1. Open `sudharma-networks/sudharma`.
2. Open **Actions**.
3. Open **Telegram Community**.
4. Tap **Run workflow**.
5. Select `mode: bootstrap`.
6. Run it against `main` after the manual-only Phase 2A PR has been merged.

Bootstrap performs only this sequence:

1. call Telegram `getMe` to verify bot identity;
2. call `getWebhookInfo`;
3. if an existing webhook URL is present, fail closed and change nothing;
4. if no webhook exists, call `deleteWebhook` with `drop_pending_updates=true` to clear pre-activation queued updates;
5. exit without creating GitHub issues and without sending Telegram replies.

A bootstrap failure must be investigated before polling. Do not blindly disable an unknown webhook.

## Controlled `/help` smoke test

After bootstrap succeeds:

1. In `@sudharma_community`, send:

```text
/help
```

2. Immediately run **Telegram Community** with `mode: poll`.
3. Verify exactly one bot reply appears in the official community group.
4. Verify the reply lists the supported commands and states that Sudharma is pre-mainnet/public-testnet experimental software.
5. Verify no GitHub issue was created from `/help`.

If this test fails, keep recurring polling disabled.

## Controlled `/report` smoke test

After `/help` succeeds, send a clearly non-sensitive test report such as:

```text
/report Phase 2A controlled test report for Telegram-to-GitHub intake verification only
```

Then:

1. Run **Telegram Community** with `mode: poll`.
2. Verify exactly one public GitHub issue is created.
3. Verify its body contains the Phase 2A source marker and a link to the Telegram test message.
4. Verify it does not contain Telegram numeric user identity data.
5. Verify the bot replies in Telegram with the new GitHub issue URL.
6. Run `mode: poll` again.
7. Verify no duplicate GitHub issue is created.

Close the controlled smoke-test issue after the evidence is recorded.

## Recurring schedule gate

The five-minute recurring poll schedule must **not** be added until all of these are true:

- Phase 2A implementation CI is green;
- bot is a normal member of `@sudharma_community` and not an admin;
- Group Privacy Mode is enabled;
- bootstrap succeeded;
- controlled `/help` succeeded;
- controlled `/report` created exactly one issue and returned its URL;
- retry/dedup behavior was verified.

Schedule activation is a separate, small PR with its own tests and exact-head CI. The intended cadence is no more frequent than every five minutes.

## Expected command responses

`/help` — supported commands and security warning.

`/rules` — concise community safety rules, including no spam/scams and no sensitive-data sharing.

`/testnet` — clearly states pre-mainnet/public-testnet status and points to official project resources.

`/miner` — points to experimental test-mining resources without profit, earnings, or investment claims.

`/report` — validates a 20–2000 code-point public report, creates/reuses the GitHub issue, and returns its URL.

## Emergency stop

For Phase 2A only, the narrowest emergency stop is:

1. remove the bot from `@sudharma_community`; or
2. disable/remove the recurring schedule if schedule activation has already been merged.

Revoking the Telegram bot token is a broader emergency action because the same token is also used by the working Phase 1 announcement bridge. Use token revocation only when the token itself is suspected to be compromised or when stopping both phases is intended.

## Troubleshooting principles

- Never paste the Telegram bot token into issues, logs, ChatGPT, screenshots, or repository files.
- Do not make the bot a group administrator as a troubleshooting shortcut.
- Do not disable Group Privacy Mode as a troubleshooting shortcut.
- A configured webhook causes bootstrap to fail by design. Identify the webhook owner/integration before changing it.
- A failed actionable Telegram update is deliberately left unacknowledged so it can be retried.
- `/report` retries search recent direct GitHub issue listings for the exact Telegram `update_id` marker before creating a new issue.

## Scope exclusions

Phase 2A does not include ordinary-chat monitoring, AI moderation, sentiment analysis, private-message support, automatic bans/mutes/deletions, attachment ingestion, personal Telegram account automation, AWS webhook infrastructure, investment promotion, mining-profitability claims, mainnet activation, unrestricted GPU mining, or GPU-PoW deployment to Seed-1/Seed-2.
