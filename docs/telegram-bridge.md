# Sudharma Telegram Bridge

Status: Phase 1 operational pre-mainnet communication tooling; Phase 2A community support is documented separately

This bridge lets the trusted Sudharma project account publish plain-text announcements to the official Telegram announcements channel through a controlled GitHub IssueOps workflow. It does **not** use a personal Telegram login and does not interact with consensus, mining activation, Seed-1/Seed-2 deployment, wallets, or mainnet.

## Security model

- Personal Telegram phone numbers, OTPs, sessions, and session databases are never used.
- `TELEGRAM_BOT_TOKEN` is a secret and must exist only in the GitHub Actions environment named `telegram-publishing`.
- Never paste the bot token into ChatGPT, a GitHub issue, a commit, a screenshot, a Telegram message, or a repository variable.
- `TELEGRAM_CHANNEL_ID` is non-secret configuration. For the public Sudharma announcements channel it can be set to `@sudharmanetworks`.
- An issue alone cannot publish. Both the issue author and the GitHub account applying the trigger label must be the trusted project account `sudharma-network`, and the label must exactly match the requested mode.
- The workflow is limited to `contents: read` and `issues: write` GitHub permissions.
- Dry-run and live publishing are separate jobs. Dry-run never receives the Telegram bot token and never calls the Telegram API.

## One-time Android setup

### 1. Create the project bot in Telegram

1. Open Telegram on Android.
2. Search for the verified `@BotFather` account.
3. Send `/newbot`.
4. Choose a display name such as `Sudharma Network Bot`.
5. Choose an available username ending in `bot`, for example `SudharmaNetworkBot` if available.
6. BotFather will return a bot token. Treat it like a password.
7. Do not send or paste that token anywhere except the GitHub Actions secret field described below.

If the token is ever exposed, use BotFather to revoke/regenerate it immediately and replace the GitHub secret.

### 2. Add the bot to the official announcements channel

1. Open the Sudharma announcements channel `@sudharmanetworks`.
2. Open channel settings.
3. Open **Administrators** and add the project bot.
4. Grant only the minimum capability needed to post channel messages for Phase 1.
5. Do not grant unrelated member-management or ownership permissions.

Removing the bot's administrator permission is the emergency stop for Phase 1 Telegram publishing.

### 3. Create the GitHub Actions environment

Use Chrome or another browser on Android if the GitHub app does not expose repository Actions settings.

1. Open `sudharma-networks/sudharma` on GitHub.
2. Open **Settings**.
3. Open **Environments**.
4. Create an environment named exactly:

```text
telegram-publishing
```

5. Inside that environment, add an environment secret named exactly:

```text
TELEGRAM_BOT_TOKEN
```

6. Paste the BotFather token only into the secret value field and save it.

Do not add the bot token as a repository variable.

### 4. Configure the official destination

In repository **Settings → Secrets and variables → Actions → Variables**, create:

```text
Name: TELEGRAM_CHANNEL_ID
Value: @sudharmanetworks
```

The destination is intentionally kept outside issue content so an issue author cannot redirect the bot to another Telegram chat.

## Creating a publish request

Create a normal GitHub issue in `sudharma-networks/sudharma`. The body must contain exactly one control marker and exactly one begin/end marker pair:

```text
<!-- sudharma-telegram-bridge:v1 -->
TELEGRAM_MESSAGE_BEGIN
Your Telegram announcement goes here.

Multiple lines are allowed.
TELEGRAM_MESSAGE_END
```

Only the text between `TELEGRAM_MESSAGE_BEGIN` and `TELEGRAM_MESSAGE_END` is eligible to be sent.

The bridge rejects:

- missing markers;
- duplicate markers;
- markers in the wrong order;
- empty messages;
- messages above 4096 Unicode code points;
- a trigger actor other than `sudharma-network`;
- an issue author other than `sudharma-network`;
- pull-request-shaped issue payloads;
- labels that do not exactly match the requested mode.

## Required operating sequence

### Step A — Dry run first

Apply the exact label:

```text
telegram:dry-run
```

A successful dry run adds a GitHub issue comment stating that validation passed, the Unicode character count, and whether the channel destination is configured. No Telegram API call is made.

If the dry run fails, correct the issue before attempting live publishing.

### Step B — Publish

After the dry-run comment confirms success, apply the exact label:

```text
telegram:publish-approved
```

The publish job validates the issue again, checks configuration, checks idempotency, claims the publish attempt, and then calls Telegram.

On success the issue receives a machine-readable marker like:

```text
<!-- sudharma-telegram-published:v1 issue=123 -->
```

The same audit comment includes the Telegram `message_id` and GitHub Actions run ID.

## Duplicate protection

The workflow serializes operations per GitHub issue and checks existing audit comments before sending.

Before the Telegram call it writes a claim marker:

```text
<!-- sudharma-telegram-publish-claim:v1 issue=123 -->
```

This protects against the dangerous case where Telegram accepted a post but GitHub failed before recording final success. If a later run finds a claim without a success marker, it fails closed instead of sending another Telegram message.

### Recovery from a claimed/uncertain attempt

1. Check the official Telegram channel manually.
2. If the message is present, do not retry; preserve the issue as an audit record and investigate why the final GitHub success comment was not written.
3. If the message is definitely absent and a retry is required, remove the stale claim comment only after that manual verification, then reapply the publish label.

This recovery is intentionally manual because preventing duplicate official announcements is safer than automatically guessing whether Telegram received an uncertain request.

## Token rotation

If the Telegram token must be rotated:

1. Open `@BotFather`.
2. Revoke/regenerate the bot token.
3. Open the GitHub `telegram-publishing` environment.
4. Replace the `TELEGRAM_BOT_TOKEN` secret value.
5. Run a dry-run issue before the next live publish.

No repository code change is required for normal token rotation.

## Emergency stop

Any one of these actions prevents new Phase 1 Telegram posts:

- remove the bot from channel administrators;
- revoke the bot token in BotFather;
- delete/disable the `TELEGRAM_BOT_TOKEN` GitHub environment secret;
- remove the `telegram:publish-approved` label from the operational process.

Revoking/deleting the shared bot token also stops Phase 2A community automation. Do not expose the token while troubleshooting.

## Phase 2A community bot

Phase 2A adds a separate, commands-only community-support path for the public group `@sudharma_community`. It does not change this Phase 1 announcement workflow or allow community content to publish into `@sudharmanetworks`.

See `docs/telegram-community-phase2a.md` for the community-bot security model, Android setup, bootstrap, smoke-test, and emergency-stop procedure.

## Phase 1 scope limits

Phase 1 supports only plain-text announcements to the official channel. It does not provide:

- personal Telegram account automation;
- group moderation;
- bans or mutes;
- inbound Telegram commands through the Phase 1 workflow;
- file or media posting;
- AWS webhook processing;
- scheduled marketing campaigns;
- investment or profitability promotion.

Phase 2A is separately scoped to explicit community commands and public GitHub report intake; broader moderation or ordinary-chat monitoring still requires separate review.

## Project safety

The Telegram bridge is communication infrastructure only. It must not activate mainnet, enable unrestricted GPU mining, deploy GPU-PoW consensus to testnet seed nodes, change monetary rules, or expose wallet/node credentials.

Public announcements should accurately describe Sudharma as pre-mainnet/public-testnet experimental software when applicable and must not promise investment returns or mining profitability.
