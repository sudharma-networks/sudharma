# Telegram Community Bot Phase 2A Design

Status: approved in chat; implementation-reviewed and hardened
Date: 2026-08-30
Project: Sudharma Network

## Purpose

Extend the existing Sudharma Telegram bridge with a narrowly scoped community bot for the public Telegram group `@sudharma_community`.

Phase 2A is command-driven community support and tester intake. It must not replace or weaken the working Phase 1 announcement path for `@sudharmanetworks`.

The bot must remain communication infrastructure only. It must not activate mainnet, enable unrestricted GPU mining, deploy GPU-PoW consensus to Seed-1/Seed-2, change monetary rules, or access wallet/node credentials.

## Chosen architecture

Use the existing Sudharma Telegram bot token stored in the GitHub Actions environment `telegram-publishing` and a GitHub Actions polling workflow.

The first production version uses GitHub Actions rather than an AWS webhook. This intentionally trades near-real-time responses for a smaller attack surface, no new always-on server, no new public webhook endpoint, and no additional credential.

Initial rollout is manual-only. After controlled bootstrap and smoke tests, a separate activation change may schedule polling approximately every five minutes. A manual `workflow_dispatch` path remains available for bootstrap and controlled tests.

### Why polling is acceptable for Phase 2A

Telegram Bot API `getUpdates` is a durable update queue. Updates are confirmed when a later call uses an offset greater than the processed `update_id`. The worker therefore does not need to store a cursor in a repository file, cache, issue, database, or AWS service.

The worker processes a returned batch in ascending `update_id` order and stops at the first actionable update that cannot be completed. It tracks the highest contiguous successfully handled or deliberately ignored update in memory, then advances Telegram's offset past only that contiguous prefix. If an actionable update fails before completion, the offset never moves past it, so Telegram can return it on the next run.

Telegram retains pending bot updates for no longer than 24 hours. Report deduplication therefore inspects a 24-hour trusted GitHub issue history, while the abuse-rate window remains independently limited to one hour.

Telegram reference: https://core.telegram.org/bots/api#getupdates

## Telegram group permissions and privacy

The bot must be added to `@sudharma_community` as a normal member, not as a group administrator.

Group Privacy Mode must remain enabled in BotFather. Telegram privacy mode is a defense-in-depth control so the bot normally receives only messages relevant to it, such as explicit commands and service messages. Making the bot a group administrator would broaden what Telegram delivers, so Phase 2A explicitly forbids group-admin status.

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

If the report command is attached to or accompanies a screenshot/video, the GitHub issue links back to the original public Telegram message. The media remains in Telegram.

### GitHub issue format and integrity

The worker creates a GitHub issue with a title derived from a short normalized prefix of the report and a structured body containing:

- a machine-readable Phase 2A Telegram `update_id` marker for retry deduplication;
- the user-provided report text;
- the Telegram source message link;
- the Telegram message timestamp in UTC;
- the ingestion timestamp in UTC;
- a note that attachments remain in Telegram;
- a pre-mainnet/public-testnet context notice.

The issue must not copy the reporter's Telegram numeric user ID or profile name. The worker neutralizes GitHub `@mention` syntax and raw HTML-comment delimiters in user-provided text so untrusted Telegram content cannot generate arbitrary GitHub mention notifications or inject a fake machine dedup marker.

Deduplication and rolling-rate state trust only qualifying report issues authored by `github-actions[bot]`. An ordinary public GitHub user cannot satisfy the worker's trusted processed-update state merely by copying a marker into an issue they created.

A successful report reply contains the created GitHub issue URL.

## Abuse controls

The community group is public, so `/report` cannot be allowed to create unbounded repository issues.

Phase 2A applies simple global intake limits rather than persisting per-user identity data:

- maximum 3 new GitHub reports processed in one polling run;
- maximum 10 trusted Telegram-created report issues in a rolling one-hour window;
- maximum 20 bot replies in one polling run, including command, throttle, validation, unknown-command, and welcome responses.

When the report limit is reached, the bot does not create another issue. It replies with a concise throttle notice and the normal GitHub Issues location so a genuine user can report manually.

These limits are operational controls, not anti-abuse guarantees. Group administrators retain normal Telegram moderation responsibility.

## Welcome behavior

Telegram service messages are allowed even with Privacy Mode enabled. Phase 2A sends one generic welcome message for a `new_chat_members` service update when the per-run reply limit permits it.

The welcome message does not mention or copy new members' names or IDs. It simply states that Sudharma is a pre-mainnet/public-testnet project and points newcomers to `/help`.

The bot does not send direct messages to new members.

## Worker components

### `scripts/telegram/community-core.js`

Pure, unit-testable logic for official-chat validation, command parsing, bot-username targeting, Unicode report validation, safe GitHub issue construction, untrusted-text neutralization, static replies, rate-limit decisions, and Telegram `update_id` marker construction/matching.

This module performs no network I/O and reads no secrets.

### `scripts/telegram/community-worker.js`

Network orchestration for one polling/bootstrap execution:

- call Telegram `getMe` to resolve the bot username;
- for bootstrap, verify webhook state and safely drop pre-activation pending updates;
- for polling, call Telegram `getUpdates`;
- filter/process updates sequentially;
- call Telegram `sendMessage` for replies;
- call GitHub REST API to create `/report` issues;
- inspect only `github-actions[bot]`-authored report issues for trusted dedup/rate state;
- use a 24-hour dedup lookup and a separate one-hour rate window;
- confirm only the contiguous successfully handled/ignored Telegram update prefix by advancing the `getUpdates` offset.

The worker uses Node's built-in `fetch` and receives credentials only through environment variables at runtime. API errors must not echo the Telegram or GitHub token.

### `.github/workflows/telegram-community.yml`

GitHub Actions entry point.

Initial rollout mode has only `workflow_dispatch` for bootstrap and smoke testing. Scheduled polling is enabled only after bootstrap and controlled live command/report tests succeed.

The job is serialized under a fixed concurrency group, uses a five-minute job timeout, and has only:

- `contents: read`
- `issues: write`

The workflow uses environment `telegram-publishing` only to obtain the already-configured `TELEGRAM_BOT_TOKEN`. It introduces no new Telegram secret or AWS dependency.

## Bootstrap and rollout safety

The bot may have pending Telegram updates before Phase 2A becomes active. Those must not suddenly create GitHub issues or replies when the worker is first enabled.

Bootstrap performs these operations in order:

1. Call Telegram `getMe` to verify the existing bot token.
2. Call `getWebhookInfo`.
3. If a webhook URL is non-empty, fail closed and make no update-mode change.
4. If no webhook is configured, call `deleteWebhook` with `drop_pending_updates=true` to clear pre-activation queued updates.
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
- Telegram reply/acknowledgment failure after GitHub issue creation: before creating a retry issue, search the previous 24 hours of trusted automation-authored report issues for the exact update marker and reuse that issue instead of creating a duplicate.
- Rate-limit/throttle decision: count only trusted report issues created during the previous hour, send the throttle response, and confirm only after the reply succeeds.
- Generic command/help replies have at-least-once delivery semantics around the narrow crash window after Telegram accepts `sendMessage` but before offset confirmation. A duplicate informational reply is preferable to losing an actionable queue position.

The worker uses GitHub's direct REST issue listing rather than eventually consistent search indexing for retry deduplication.

## Testing strategy

All new logic is developed test-first.

Unit/integration tests cover at minimum:

- official group accepted; other groups/private chats rejected;
- ordinary messages ignored;
- `/help`, `/rules`, `/testnet`, `/miner`, `/report` parsing;
- bot-targeted command parsing and other-bot rejection;
- report min/max Unicode lengths;
- GitHub mention and raw marker-injection neutralization;
- report body update marker and Telegram source link;
- no Telegram user numeric ID/profile identity in GitHub issue content;
- production trusted-state filtering to `github-actions[bot]`;
- 24-hour retry deduplication and independent one-hour abuse counting;
- per-run report and reply caps;
- generic welcome service-message behavior;
- retry lookup by exact `update_id` marker;
- offset confirmation only across a contiguous successfully handled/ignored prefix.

Workflow contract tests verify:

- no `contents: write` permission;
- environment secret is referenced but never printed;
- initial rollout has no recurring schedule;
- poll runs are serialized;
- each job is bounded by a five-minute timeout;
- Phase 1 announcement workflow remains present and operational.

Repository CI continues to run the full existing Go/testnet/container/race suite in addition to Telegram Node tests.

## Deliberate exclusions

Phase 2A does not include ordinary-chat monitoring, AI moderation or sentiment analysis, bans/mutes/message deletion, attachment ingestion, private-message support, personal Telegram account/session automation, AWS webhook infrastructure, investment or profitability promotion, mainnet activation, unrestricted GPU mining, or GPU-PoW deployment to Seed-1/Seed-2.

## Future Phase 2B options

After Phase 2A operates reliably, the same pure command/report logic can move behind an AWS Lambda Telegram webhook for near-real-time responses without changing command semantics.

Additional features such as structured multi-step report forms, media attachment ingestion, opt-in FAQ buttons, or narrowly scoped moderation can be designed separately. They are not prerequisites for Phase 2A.

## Success criteria

Phase 2A is complete only when:

- the bot responds to supported explicit commands in `@sudharma_community`;
- ordinary conversation does not trigger bot behavior;
- generic welcome behavior works without exposing member identity data;
- `/report` creates one structured GitHub issue and replies with its URL;
- retries do not create duplicate report issues across Telegram's 24-hour pending-update lifetime;
- public GitHub users cannot spoof trusted processed-update state;
- report and reply abuse limits work;
- no personal Telegram session or new secret is introduced;
- the bot has no group-admin/moderation powers;
- the scheduled poller is enabled only after controlled bootstrap and smoke testing;
- existing Phase 1 announcement publishing remains operational;
- full repository CI is green;
- no mainnet/mining/seed-node safety gate changes occur.
