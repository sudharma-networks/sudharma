# Telegram Community Bot Phase 2A Design

Status: approved in chat; written-spec review pending
Date: 2026-08-30
Project: Sudharma Network

## Purpose

Extend the existing Sudharma Telegram bridge with a narrowly scoped community bot for the public Telegram group `@sudharma_community`.

Phase 2A is command-driven community support and tester intake. It must not replace or weaken the working Phase 1 announcement path for `@sudharmanetworks`.

The bot must remain communication infrastructure only. It must not activate mainnet, enable unrestricted GPU mining, deploy GPU-PoW consensus to Seed-1/Seed-2, change monetary rules, or access wallet/node credentials.

## Chosen architecture

Use the existing Sudharma Telegram bot token stored in the GitHub Actions environment `telegram-publishing` and a GitHub Actions polling workflow.

The first production version uses GitHub Actions rather than an AWS webhook. This intentionally trades near-real-time responses for a smaller attack surface, no new always-on server, no new public webhook endpoint, and no additional credential.

Once enabled, the scheduled workflow polls approximately every five minutes. A manual `workflow_dispatch` path remains available for bootstrap and controlled smoke tests.

### Why polling is acceptable for Phase 2A

Telegram Bot API `getUpdates` is a durable update queue. Updates are confirmed when a later call uses an offset greater than the processed `update_id`. The worker therefore does not need to store a cursor in a repository file, cache, issue, database, or AWS service.

The worker processes a returned batch in ascending `update_id` order and stops at the first actionable update that cannot be completed. It tracks the highest contiguous successfully handled or deliberately ignored update in memory, then advances Telegram's offset past only that contiguous prefix. If an actionable update fails before completion, the offset never moves past it, so Telegram can return it on the next run.

This design is intentionally simple and auditable. It avoids GitHub-side cursor state and keeps GitHub permissions focused on the only persistent side effect Phase 2A needs: issue creation for tester reports.

Telegram reference: https://core.telegram.org/bots/api#getupdates

## Telegram group permissions and privacy

The bot must be added to `@sudharma_community` as a normal member, not as a group administrator.

Group Privacy Mode must remain enabled in BotFather. Telegram privacy mode is a defense-in-depth control so the bot normally receives only messages relevant to it, such as explicit commands and replies, plus service messages. Making the bot a group administrator would broaden what Telegram delivers, so Phase 2A explicitly forbids group-admin status.

The worker also enforces its own application-level filter. It ignores all updates unless the chat is the official public group with username exactly `sudharma_community` and the chat type is `group` or `supergroup`.

The bot receives no moderation permissions. Phase 2A cannot ban, mute, delete messages, change group information, add administrators, manage invite links, or inspect Telegram account sessions.

Telegram privacy reference: https://core.telegram.org/bots/features#privacy-mode

## Supported inputs

Phase 2A recognizes only explicit bot commands at the beginning of a Telegram message or caption.

Supported commands:

- `/help` — show the Phase 2A command list and safety notice.
- `/rules` — show concise community safety rules.
- `/testnet` — explain that Sudharma is pre-mainnet/public-testnet experimental software and point to official project resources.
- `/miner` — point to test-mining resources and feedback paths without profitability or investment claims.
- `/report` — create a structured public GitHub issue from a tester/user problem report.

The parser accepts Telegram's normal command form and the explicit form addressed to this bot, for example `/help@BotUsername`. If a command is addressed to another bot username, it is ignored.

An unknown explicit command addressed to this bot receives one concise `/help` response and never triggers privileged actions.

Ordinary group conversation is ignored even if Telegram delivers it unexpectedly.

## `/report` behavior

A report is accepted only when text follows `/report` in the same message or caption.

Recommended usage:

`/report miner exits after startup on my RX 580; OpenCL initialization error shown in log`

Validation rules:

- empty reports are rejected with usage guidance;
- report text must be at least 20 Unicode code points;
- report text must be no more than 2000 Unicode code points;
- the bot reminds users that accepted report text is mirrored into the project's public GitHub repository and that secrets, seed phrases, private keys, passwords, tokens, personal information, and other sensitive data must not be included;
- attachments are not copied to GitHub in Phase 2A.

If the report command is attached to or replies around a screenshot/video, the GitHub issue links back to the original public Telegram message. The media remains in Telegram.

### GitHub issue format

The worker creates a GitHub issue with a title derived from a short normalized prefix of the report and a structured body containing:

- a Phase 2A source marker;
- a machine-readable Telegram `update_id` marker for retry deduplication;
- the user-provided report text;
- the Telegram source message link;
- the Telegram message timestamp in UTC;
- a note that attachments remain in Telegram;
- a pre-mainnet/public-testnet context notice.

The issue must not copy the reporter's Telegram numeric user ID. The worker also neutralizes GitHub `@mention` syntax in user-provided report text so untrusted Telegram content cannot generate arbitrary GitHub mention notifications.

A successful report reply contains the created GitHub issue URL.

## Abuse controls

The community group is public, so `/report` cannot be allowed to create unbounded repository issues.

Phase 2A applies simple global intake limits rather than persisting per-user identity data:

- maximum 3 new GitHub reports processed in one polling run;
- maximum 10 Telegram-created report issues in a rolling one-hour window;
- maximum 20 bot replies in one polling run, including command, throttle, validation, unknown-command, and welcome responses.

When the report limit is reached, the bot does not create another issue. It replies with a concise throttle notice and the normal GitHub Issues location so a genuine user can report manually.

These limits are operational controls, not anti-abuse guarantees. Group administrators retain normal Telegram moderation responsibility.

## Welcome behavior

Telegram service messages are allowed even with Privacy Mode enabled. Phase 2A sends one generic welcome message for a `new_chat_members` service update when the per-run reply limit permits it.

The welcome message does not mention or copy new members' names or IDs. It simply states that Sudharma is a pre-mainnet/public-testnet project and points newcomers to `/help`.

The bot does not send direct messages to new members.

## Worker components

### `scripts/telegram/community-core.js`

Pure, unit-testable logic for:

- official-chat validation;
- command parsing;
- bot-username targeting;
- Unicode length validation;
- safe GitHub issue title/body construction;
- GitHub mention neutralization;
- static command/welcome response construction;
- report-rate decision helpers;
- Telegram `update_id` marker construction and matching.

This module performs no network I/O and reads no secrets.

### `scripts/telegram/community-worker.js`

Network orchestration for one polling/bootstrap execution:

- call Telegram `getMe` to resolve the bot username;
- for bootstrap, verify webhook state and safely drop pre-activation pending updates;
- for polling, call Telegram `getUpdates`;
- filter/process updates sequentially;
- call Telegram `sendMessage` for replies;
- call GitHub REST API to create `/report` issues, count recent Telegram-created reports, and find an existing issue for retry deduplication;
- confirm only the contiguous successfully handled/ignored Telegram update prefix by advancing the `getUpdates` offset.

The worker uses Node's built-in HTTP/fetch APIs and receives credentials only through environment variables at runtime.

### `.github/workflows/telegram-community.yml`

GitHub Actions entry point.

Initial rollout mode has `workflow_dispatch` for bootstrap and smoke testing. Scheduled polling is enabled only after bootstrap and controlled live command/report tests succeed.

Final scheduled mode runs no more frequently than every five minutes and uses fixed concurrency so two pollers cannot process the same queue at the same time.

Required permissions:

- `contents: read`
- `issues: write`

The workflow uses environment `telegram-publishing` only to obtain the already-configured `TELEGRAM_BOT_TOKEN`. It introduces no new secret.

## Bootstrap and rollout safety

The bot may have pending Telegram updates before Phase 2A becomes active. Those must not suddenly create GitHub issues or replies when the worker is first enabled.

Bootstrap performs these operations in order:

1. Call Telegram `getMe` to verify the existing bot token.
2. Call `getWebhookInfo`.
3. If a webhook URL is non-empty, fail closed and make no update-mode change. This prevents Phase 2A from silently disabling an unknown integration.
4. If no webhook is configured, call `deleteWebhook` with `drop_pending_updates=true`. Telegram documents this as a supported way to clear pending updates when switching/ensuring `getUpdates` mode.
5. Exit without creating GitHub issues and without sending Telegram replies.

Telegram bootstrap references:

- https://core.telegram.org/bots/api#getwebhookinfo
- https://core.telegram.org/bots/api#deletewebhook

Production schedule activation happens only after:

1. Phase 2A code and tests are green.
2. The bot is added to `@sudharma_community` as a normal member, not admin.
3. BotFather Group Privacy Mode is verified enabled.
4. Bootstrap completes successfully.
5. A controlled `/help` command is processed successfully in the official community group.
6. A controlled `/report` smoke test creates exactly one GitHub issue and returns its URL.
7. Retry/deduplication behavior is reviewed before recurring polling is enabled.

## Telegram reply behavior

Replies are plain text with no Telegram `parse_mode`, preventing Telegram markup injection from user content.

Where supported, replies reference the triggering message and preserve `message_thread_id` for forum-topic compatibility.

Link previews are disabled for bot replies unless a future feature explicitly needs them.

The bot never posts user-supplied text to the announcements channel. Phase 1 announcement publishing remains a separate workflow and authorization path.

## Error handling and delivery semantics

- Telegram/API/network failure before an update is completed: do not advance the offset past that update.
- Invalid or out-of-scope chat/message: deliberately ignore it and confirm the update so it cannot block the queue.
- Unknown explicit bot command: send concise help, then confirm only after the reply succeeds.
- Invalid `/report`: send usage/privacy guidance, then confirm only after the reply succeeds.
- GitHub issue creation failure: do not confirm that report update; retry on a future poll.
- Telegram acknowledgment failure after GitHub issue creation: before creating a retry issue, detect the existing Phase 2A update marker in recent GitHub issues and reuse that issue instead of creating a duplicate.
- Rate-limit/throttle decision: send the throttle response and confirm only after the reply succeeds.
- Generic command/help replies have at-least-once delivery semantics around an extremely narrow crash window after Telegram accepts `sendMessage` but before `getUpdates` offset confirmation. A duplicate informational reply is preferable to losing an actionable queue position. `/report` issue creation has explicit retry deduplication because duplicate public issues are more harmful.

On a `/report` retry, the worker inspects recent Phase 2A report issues directly through GitHub's REST issue listing rather than relying on eventually consistent search indexing. This provides practical duplicate protection around the GitHub-create/Telegram-reply boundary.

## Testing strategy

All new logic is developed test-first.

Unit tests cover at minimum:

- official group accepted; other groups/private chats rejected;
- ordinary messages ignored;
- `/help`, `/rules`, `/testnet`, `/miner`, `/report` parsing;
- bot-targeted command parsing;
- commands addressed to another bot rejected;
- unknown explicit bot command maps to help;
- report min/max Unicode lengths;
- GitHub mention neutralization;
- report body source marker, `update_id` marker, and message link;
- no Telegram user numeric ID in GitHub issue content;
- per-run and rolling-hour rate-limit decisions;
- generic welcome service-message behavior and reply-cap accounting;
- retry lookup by `update_id` marker;
- offset confirmation advances only across a contiguous successfully handled/ignored prefix.

Workflow contract tests verify:

- no `contents: write` permission;
- environment secret is referenced but never printed;
- official group is pinned in code/config rather than accepted from Telegram input;
- bootstrap checks webhook state before dropping pending updates;
- bootstrap cannot create issues or send replies;
- initial rollout has no recurring schedule until smoke tests pass;
- poll runs are serialized with concurrency;
- Phase 1 announcement workflow remains unchanged by Phase 2A.

Repository CI continues to run the full existing Go/testnet/container/race suite in addition to Telegram Node tests.

## Deliberate exclusions

Phase 2A does not include:

- reading or classifying ordinary conversation;
- AI moderation or automatic sentiment analysis;
- bans, mutes, message deletion, or admin actions;
- downloading/copying Telegram photos, videos, documents, voice notes, or files to GitHub;
- private-message support;
- personal Telegram account/session automation;
- AWS webhook infrastructure;
- investment, price, earnings, or mining-profitability promotion;
- mainnet or GPU consensus deployment changes.

## Future Phase 2B options

After Phase 2A operates reliably, the same pure command/report logic can move behind an AWS Lambda Telegram webhook for near-real-time responses without changing command semantics.

Additional features such as structured multi-step report forms, media attachment ingestion, opt-in FAQ buttons, or narrowly scoped moderation can be designed separately. They are not prerequisites for Phase 2A.

## Success criteria

Phase 2A is complete only when:

- the bot responds to supported explicit commands in `@sudharma_community`;
- ordinary conversation does not trigger bot behavior;
- generic welcome behavior works without exposing member identity data;
- `/report` creates one structured GitHub issue and replies with its URL;
- retries do not create duplicate report issues;
- report and reply abuse limits work;
- no personal Telegram session or new secret is introduced;
- the bot has no group-admin/moderation powers;
- the scheduled poller is enabled only after controlled bootstrap and smoke testing;
- existing Phase 1 announcement publishing remains operational;
- full repository CI is green;
- no mainnet/mining/seed-node safety gate changes occur.
