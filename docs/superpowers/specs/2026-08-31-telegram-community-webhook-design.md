# Telegram Community Webhook — Design

Date: 2026-08-31
Status: proposed design, not deployed

## Problem

The current Sudharma Telegram community bot is implemented as a scheduled GitHub Actions poller. The workflow runs every five minutes and invokes the worker in `poll` mode. The worker calls Telegram `getUpdates` with `timeout: 0`, processes queued updates, replies, and exits. This architecture introduces up to roughly one schedule interval of command latency, plus GitHub scheduler and runner startup delay.

The goal is to make supported Telegram community commands and new-member welcomes respond promptly while preserving the existing safety model and keeping live blockchain infrastructure unchanged.

## Scope

This design changes only the interactive Telegram community delivery path for `@sudharma_community`.

In scope:

- `/help`
- `/rules`
- `/testnet`
- `/miner`
- `/report <problem>`
- new-member welcome events
- authenticated Telegram webhook delivery
- reuse of existing command parsing, safety text, report validation, rate limits, deduplication, and GitHub issue creation
- controlled cutover from scheduled polling to webhook operation
- rollback to the existing polling path

Out of scope:

- moderation powers
- ordinary-chat monitoring
- private-message support
- AI moderation
- consensus, mining activation, seed-node control, faucet behavior, wallet signing, or blockchain parameters
- changes to Phase 1 announcement publishing
- automatic mainnet or testnet deployment

## Existing safety properties to preserve

The webhook path must preserve the current Phase 2A guarantees:

1. The bot remains a normal member of `@sudharma_community`, not an administrator.
2. BotFather Group Privacy Mode remains enabled.
3. Only messages from chat username `sudharma_community` and chat type `group` or `supergroup` are actionable.
4. Only supported explicit commands and new-member service updates are handled.
5. Ordinary conversation remains ignored.
6. Commands addressed to another bot remain ignored.
7. Community-provided text is never forwarded into the official announcements channel.
8. `/report` keeps its Unicode length checks, sensitive-data warning, GitHub mention neutralization, HTML-comment neutralization, deduplication, and abuse limits.
9. Telegram numeric user IDs and profile names are not copied into GitHub issues.
10. Secrets never appear in source, logs, test fixtures, issue bodies, or user-facing errors.

## Architecture

### Request flow

```text
Telegram Bot API
    |
    | HTTPS POST + X-Telegram-Bot-Api-Secret-Token
    v
API Gateway HTTPS endpoint
    |
    v
Dedicated Telegram Community Lambda
    |
    +--> verify webhook secret token
    +--> parse Telegram update
    +--> reuse community-core command classification
    +--> send immediate Telegram reply
    +--> optionally create/reuse GitHub report issue
```

The Lambda is dedicated to Telegram community interaction. It must not share execution paths with blockchain consensus, mining, faucet signing, wallet key handling, or seed-node administration.

## Webhook authentication

Telegram webhook registration will use Telegram's `secret_token` option. Telegram then sends the configured value in the `X-Telegram-Bot-Api-Secret-Token` request header.

The Lambda must:

1. require the secret-token header on every request;
2. compare it to a configured secret using a timing-safe comparison;
3. reject missing or mismatched values before parsing or acting on the Telegram update;
4. return a generic authorization failure without logging either supplied or expected secret values.

The webhook secret must be distinct from the Telegram bot token.

## Secret handling

The Telegram bot token and webhook secret must be stored in protected AWS secret/configuration storage and injected at runtime. No secret values may be committed to GitHub.

The Lambda should receive only the minimum configuration it needs:

- Telegram bot token
- Telegram webhook secret
- GitHub repository identifier
- GitHub authentication material required for `/report`, if `/report` remains synchronous in Lambda

Any GitHub credential used by Lambda must be scoped to the minimum permissions necessary to inspect/create issues in the Sudharma repository. If a suitable short-lived or brokered credential path is available, prefer it over a long-lived personal token.

## Reuse of existing code

The existing `scripts/telegram/community-core.js` remains the source of truth for:

- official chat validation
- command parsing
- static replies
- report validation
- report issue construction
- deduplication markers
- per-run/rolling abuse decisions

Webhook-specific code should be a thin adapter around this logic rather than a second implementation of commands.

Where worker-only logic is currently embedded in the polling worker, extract only the smallest reusable units needed for webhook execution. Avoid unrelated refactoring.

## Processing model

### Informational commands

For `/help`, `/rules`, `/testnet`, `/miner`, and new-member welcome events:

1. authenticate webhook request;
2. parse JSON update;
3. validate official community chat;
4. build existing static reply;
5. send Telegram response;
6. return HTTP success.

The target is human-perceived near-immediate response. No GitHub Actions runner is involved.

### `/report`

`/report` keeps existing validation and deduplication semantics.

Preferred initial implementation: process `/report` synchronously in Lambda because current behavior is already bounded and straightforward. The Lambda must use short network timeouts for Telegram and GitHub calls and fail without acknowledging an operation as completed when issue creation cannot be verified.

If measured latency or reliability later shows synchronous GitHub issue creation is unsuitable, queueing can be designed as a separate follow-up project. It is not part of this initial migration.

## Idempotency and duplicate delivery

Telegram may retry webhook deliveries. The handler must therefore be safe for duplicate `update_id` delivery.

For informational commands and welcomes, duplicate webhook delivery should not produce repeated replies within a reasonable idempotency window. The implementation should persist processed Telegram `update_id` values in a small durable store with TTL.

For `/report`, preserve the existing issue-marker deduplication as a second layer. A Telegram retry must not create multiple GitHub issues for the same update.

Recommended durable idempotency store: a dedicated DynamoDB table keyed by Telegram `update_id`, with TTL. This store contains only update identifiers and minimal processing state; it must not store Telegram profile data or message contents unless required for a narrowly justified recovery path.

## Ordering and concurrency

The current poller processes updates in ascending `update_id` order. Webhook delivery changes the concurrency model because requests may arrive independently.

For the supported command-only workload, strict total ordering is not required for informational commands. Idempotency is required.

For `/report`, duplicate suppression and rate limiting must remain correct under concurrent Lambda invocations. Rate-limit state must not rely on in-memory counters alone. Any new DynamoDB state updates must use atomic conditional operations.

## Rate limits

Preserve existing abuse controls in equivalent or stricter form:

- maximum 10 Telegram-created reports in a rolling one-hour window;
- duplicate report prevention by `update_id`;
- bounded outbound Telegram sends;
- bounded request execution time.

The existing 'maximum 3 reports per poll run' concept is tied to batch polling and does not map directly to one-update-per-webhook invocation. Replace it with a webhook-appropriate burst control during implementation while keeping or strengthening the current effective abuse resistance.

The implementation plan must define and test the exact burst policy before deployment.

## API Gateway and Lambda boundaries

API Gateway should expose one dedicated POST route for Telegram webhook delivery.

Requirements:

- HTTPS only;
- no public diagnostic endpoint that leaks configuration;
- request body size bounded to what Telegram updates require;
- Lambda execution timeout kept short;
- no VPC access unless a concrete dependency requires it;
- least-privilege IAM role;
- logs sanitized and retention explicitly configured;
- no request-body logging at API Gateway or Lambda because Telegram updates may contain user-provided content.

## Observability

Logs may contain only operational metadata such as:

- invocation outcome category;
- Telegram `update_id`;
- command type after classification;
- sanitized external-service diagnostic code;
- duration bucket or numeric latency;
- whether an event was ignored, handled, duplicated, throttled, or failed.

Logs must not include:

- bot token;
- webhook secret;
- seed phrases/private keys/passwords/tokens;
- Telegram profile names or numeric user IDs;
- full message/report text;
- raw Telegram API error descriptions that may echo sensitive input.

## Deployment and cutover

Development and testing do not alter the live webhook or current polling schedule.

Deployment is a separate gated operation after implementation CI and review.

Safe cutover sequence:

1. deploy Lambda/API Gateway with no Telegram webhook configured;
2. verify endpoint locally/integration tests using synthetic authenticated requests;
3. verify production endpoint health with a non-Telegram synthetic request path only if one can be done without exposing secrets;
4. read current Telegram `getWebhookInfo` and confirm expected pre-cutover state;
5. temporarily disable scheduled polling to prevent dual consumers;
6. register the new Telegram webhook with the secret token;
7. verify `getWebhookInfo` points exactly to the expected endpoint and shows no unexpected errors;
8. run controlled `/help` smoke test;
9. run controlled `/report` smoke test with non-sensitive content;
10. observe latency and duplicate behavior;
11. keep the old poller available for rollback but disabled.

Do not run webhook and `getUpdates` polling concurrently. Telegram treats webhook delivery and long polling as mutually exclusive consumption modes.

## Rollback

Rollback must be possible without changing the bot token.

Sequence:

1. disable/delete the Telegram webhook without dropping pending updates unless specifically justified;
2. verify `getWebhookInfo` reports no active webhook;
3. re-enable the known-good scheduled polling workflow;
4. perform controlled `/help` smoke test;
5. investigate webhook failure offline.

If the bot token itself is suspected compromised, token rotation is a separate emergency procedure affecting both community and announcement paths.

## Testing requirements

Implementation must include automated tests covering at least:

- valid webhook secret accepted;
- missing webhook secret rejected;
- incorrect webhook secret rejected;
- secret comparison does not expose values in errors/logs;
- malformed JSON rejected safely;
- non-Telegram/unexpected payload ignored or rejected safely;
- other groups/private chats remain ignored;
- commands addressed to other bots remain ignored;
- `/help`, `/rules`, `/testnet`, `/miner` preserve existing replies;
- approved full welcome copy remains unchanged, including official website URL;
- new-member update receives one welcome reply;
- duplicate informational `update_id` does not produce duplicate reply;
- `/report` validation unchanged;
- `/report` duplicate delivery creates no duplicate GitHub issue;
- concurrent rate-limit handling is atomic;
- Telegram API timeout/failure handled safely;
- GitHub API timeout/failure handled safely;
- logs do not reveal message text, profile details, tokens, secrets, or credentials;
- existing Phase 1 Telegram announcement tests remain green;
- existing Phase 2A command behavior tests remain green or are intentionally migrated with equivalent assertions.

## Acceptance criteria

The change is ready for a separate deployment decision only when:

1. all repository CI and Telegram-specific tests pass at the exact reviewed head;
2. security review finds no secret leakage or privilege expansion beyond the documented path;
3. API Gateway/Lambda configuration is reproducible and documented;
4. rollback procedure is documented and tested without touching blockchain services;
5. no live webhook has been activated during ordinary development;
6. controlled post-deployment `/help` response is observed within a target of 5 seconds under normal conditions;
7. `/report` creates exactly one issue and returns exactly one Telegram response;
8. duplicate webhook delivery does not duplicate replies or issues;
9. the existing public testnet, faucet, explorer, wallet, miners, and seeds remain unchanged by this work.

## Open implementation decision

The implementation plan must choose the exact AWS provisioning mechanism already acceptable for this repository (for example, a minimal scripted AWS CLI deployment or repository-standard infrastructure definition). No established Lambda/IaC pattern was found in the repository search used for this design, so the implementation must not invent a broad infrastructure framework solely for this feature.

## Safety gate

This document approves design only. It does not authorize live AWS mutation or Telegram webhook activation. Live cutover requires a separate explicit deployment approval after implementation, tests, exact-head CI, review, and rollback verification.
