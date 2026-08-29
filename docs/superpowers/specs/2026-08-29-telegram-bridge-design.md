# Sudharma Telegram Bridge — Design

Date: 2026-08-29
Status: Approved architecture, pre-implementation
Target branch: `feature/telegram-bridge`

## Goal

Allow the Sudharma project to publish official Telegram announcements through a controlled GitHub workflow without exposing a personal Telegram account, Telegram login session, OTP, or bot token to ChatGPT or to repository content.

Phase 1 is intentionally narrow: official channel publishing only. Community moderation, inbound webhooks, scheduled marketing, and automated replies are deferred until this path is proven safe and reliable.

## Why this approach

ChatGPT does not currently have a native Telegram connector in this environment. The project already uses GitHub Actions and the repository is connected to ChatGPT with authorized GitHub access, so GitHub is the safest existing control plane.

Telegram provides an HTTPS Bot API. The bot token is the authentication credential and must remain secret. GitHub Actions supports repository or environment secrets, which allows the workflow to use the token without storing it in source code.

## Architecture

Control flow:

1. A maintainer or authorized ChatGPT/GitHub action creates a specially formatted GitHub issue containing the proposed Telegram message.
2. The issue is marked with a maintainer-controlled label such as `telegram:publish-approved`.
3. A GitHub Actions workflow reacts to the label event.
4. The workflow verifies that the issue author is trusted, the required control marker is present, the message is valid, and the destination is the configured Sudharma channel.
5. The workflow reads `TELEGRAM_BOT_TOKEN` from a GitHub Actions environment secret and `TELEGRAM_CHANNEL_ID` from repository/environment configuration.
6. The workflow sends the message to Telegram over HTTPS using the Telegram Bot API.
7. The workflow records success or failure back on the GitHub issue, including the returned Telegram message ID on success.

The destination is never supplied by an issue author. It is configured separately so an attacker cannot redirect the bot to another chat.

## Security boundaries

### Personal Telegram account

The owner's personal Telegram login, phone number, OTP, session, and session database are never used by the bridge.

### Bot token

`TELEGRAM_BOT_TOKEN` must exist only as a GitHub Actions secret. It must never be committed, placed in issue text, printed to logs, pasted into ChatGPT, or stored in repository variables.

If the token is ever exposed, it must be revoked and regenerated in BotFather immediately.

### Telegram destination

The bridge posts only to the official Sudharma announcements channel configured in `TELEGRAM_CHANNEL_ID`. The issue payload cannot override this value.

### GitHub trigger authorization

Opening an issue alone is never sufficient to publish. The workflow requires all of the following:

- the expected control marker in the issue body;
- the exact publish-approval label;
- a trusted GitHub author association such as OWNER, MEMBER, or COLLABORATOR;
- a valid non-empty message within Telegram limits;
- the configured GitHub Actions environment containing the Telegram secret.

The publish label is intended to be maintainer-controlled. Public contributors can propose content, but cannot cause it to be published merely by filing an issue.

### Workflow permissions

Use least privilege. The workflow needs only what is required to read issue data and write an audit comment/status. It does not need AWS credentials, package publishing rights, repository content write access, mainnet credentials, wallet keys, miner keys, or consensus deployment permissions.

## Phase 1 functionality

### Supported

- plain-text Telegram channel announcements;
- line breaks and normal URLs;
- controlled publish action from GitHub IssueOps;
- dry-run validation path that does not contact Telegram;
- audit comment containing Telegram message ID after successful publish;
- idempotency protection so the same approval event cannot publish twice accidentally;
- clear failure comments without exposing secrets.

### Deferred

- editing previously published messages;
- deleting messages;
- posting media/files;
- Telegram community-group moderation;
- receiving Telegram messages or bug reports;
- AWS Lambda webhook receiver;
- scheduled posting;
- automated replies;
- user banning/muting;
- cryptocurrency price or investment promotion.

Editing and deletion can be added as a small Phase 1B after publishing is stable.

## Issue command format

The bridge will use a strict marker pair instead of trying to interpret arbitrary Markdown.

Example:

```text
<!-- sudharma-telegram-bridge:v1 -->
TELEGRAM_MESSAGE_BEGIN
Public testnet update...
TELEGRAM_MESSAGE_END
```

Only text inside the marker pair is sent. The parser rejects missing markers, duplicate marker pairs, empty content, oversized messages, and malformed control blocks.

## Dry run

Before the bot token is added, the same parser and authorization checks can run under a `telegram:dry-run` label. A dry run must never call `api.telegram.org` and must report the validated character count and destination configuration state without exposing the token.

This provides a safe way to verify the GitHub side before Telegram credentials are introduced.

## Idempotency

A successful publish writes a machine-readable success marker to the issue comment history containing the issue number and Telegram message ID. Before publishing, the workflow checks for an existing success marker for the same issue.

If one exists, the workflow exits successfully without sending again.

This protects against label reapplication, manual reruns, or duplicate event delivery.

## Error handling

The workflow fails closed.

It must not publish when:

- the bot token is missing;
- the destination is missing;
- authorization checks fail;
- the issue payload is malformed;
- the message is empty or too long;
- Telegram returns an error;
- the response cannot be parsed safely.

Failure output must not include the bot token or full request URL containing the token.

## Logging and audit

GitHub issue comments provide the human-readable audit trail. GitHub Actions logs provide execution-level diagnostics.

Success records include:

- action type;
- GitHub issue number;
- Telegram message ID;
- timestamp;
- workflow run reference.

No secret values are logged.

## Telegram bot permissions

For Phase 1 the bot should receive only the minimum channel administrator capability needed to publish channel posts. Broader permissions such as member banning, invitation management, ownership transfer, or unrelated group administration are not required.

When Phase 1B editing/deletion is implemented, add only the additional Telegram permissions actually required for those operations.

## Repository changes expected during implementation

Likely files:

- `.github/workflows/telegram-publish.yml` — event gate and publish workflow;
- `scripts/telegram/parse-command.js` — deterministic issue-body parser and validation;
- `scripts/telegram/parse-command.test.js` — parser/security tests;
- `docs/telegram-bridge.md` — operator instructions, Android setup, recovery, and token-rotation guide.

No consensus, miner, wallet-key, mainnet, seed-node, or AWS deployment code should be modified for Phase 1.

## Testing strategy

Implementation follows test-first development.

Required automated cases include:

1. valid message parses exactly;
2. missing begin marker is rejected;
3. missing end marker is rejected;
4. duplicate marker blocks are rejected;
5. empty message is rejected;
6. oversized message is rejected;
7. whitespace and line breaks are preserved;
8. untrusted issue context cannot reach the publish step;
9. dry-run cannot call Telegram;
10. already-published issue is detected and skipped;
11. Telegram non-2xx/API error fails the workflow;
12. secret values are never intentionally echoed.

A live smoke test is performed only after the owner creates the bot, adds it to the official channel, and stores the token as a GitHub secret. The first live message should be an explicit test announcement that can be deleted manually if necessary.

## Android-only owner setup

The owner can complete the one-time Telegram and GitHub configuration from Android:

1. create the project bot with BotFather;
2. add the bot as an administrator of the official Sudharma announcement channel;
3. grant only the posting permission needed for Phase 1;
4. open the Sudharma GitHub repository settings in a mobile browser;
5. add the bot token as `TELEGRAM_BOT_TOKEN` in the dedicated Telegram GitHub Actions environment;
6. configure the official channel identifier separately as `TELEGRAM_CHANNEL_ID`;
7. never paste the bot token into ChatGPT, an issue, commit, screenshot, or public message.

## Safety and project scope

This subsystem is communication infrastructure only. It does not activate mainnet, deploy GPU-PoW consensus, enable unrestricted mining, alter wallet keys, change supply rules, or modify Seed-1/Seed-2 consensus behavior.

Public messages should continue to describe Sudharma as pre-mainnet/public-testnet experimental software where appropriate and must not promise investment returns or mining profitability.

## Future Phase 2

After Phase 1 is stable, a separate design can add inbound Telegram community functionality using an AWS Lambda webhook with Telegram's webhook secret-token verification. That phase may route structured bug reports or hardware-test submissions to GitHub, but it will be reviewed separately because receiving untrusted public input creates a larger attack surface.

## Success criteria

Phase 1 is complete when:

- repository tests pass;
- dry-run IssueOps works without any Telegram credential;
- the bot token is stored only in GitHub Secrets;
- a trusted, approved GitHub issue publishes exactly one message to the official channel;
- a repeat event does not duplicate the message;
- GitHub records the returned Telegram message ID;
- unauthorized or malformed requests fail closed;
- no protected Sudharma consensus/mainnet systems are touched.
