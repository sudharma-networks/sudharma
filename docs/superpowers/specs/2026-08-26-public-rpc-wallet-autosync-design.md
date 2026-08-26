# Sudharma Public RPC Wallet Autosync Design

## Goal

Provide the Android Testnet wallet with automatic, secure, public synchronization from anywhere without asking ordinary users to configure an RPC address, while keeping wallet keys and signing exclusively on-device and keeping raw node RPC private.

## Public architecture

The Android Testnet wallet uses a stable AWS API Gateway HTTP API `execute-api` HTTPS endpoint. API Gateway invokes a small Node.js Lambda function attached to the Sudharma Testnet VPC. Lambda proxies only an explicit wallet-safe route allowlist to the two private seed-side Nginx listeners on TCP 29100. Those listeners in turn proxy only approved routes to each node's localhost RPC at `127.0.0.1:28545`.

The seed nodes remain independent and synchronized. Raw RPC is never exposed publicly. Lambda uses both seeds for availability. Read requests fail over between seeds when the preferred seed is unavailable. A transaction retry may resend only the exact same signed transaction bytes and therefore the same deterministic transaction ID; it must never create, re-sign, mutate, or substitute a transaction.

## Public route allowlist

Only these operations are public through API Gateway/Lambda:

- `GET /health`
- `GET /ready`
- `GET /v1/status`
- `GET /v1/accounts/{address}`
- `POST /v1/transactions`
- `GET /v1/transactions/{transactionID}`

The public path must not expose `/metrics`, raw RPC, administrative operations, block enumeration, mempool enumeration, or any catch-all proxy behavior.

## Seed-side boundary

Each seed exposes a private Nginx listener on TCP 29100 bound to its private VPC address only. The listener applies method restrictions, request-size limits for transaction submission, bounded upstream timeouts, `Cache-Control: no-store`, and a default 404 for all routes outside the allowlist.

Current private endpoints:

- Seed-1: `http://172.31.10.171:29100`
- Seed-2: `http://172.31.32.195:29100`

Raw node RPC remains `127.0.0.1:28545` on each seed.

## Lambda behavior

Lambda receives API Gateway HTTP API events and normalizes method/path/body without logging sensitive transaction bodies. It validates the route and basic path shape before making any upstream request. Requests are bounded by body size and timeout.

For read operations, Lambda attempts a preferred healthy seed and fails over to the second seed on connection errors, timeouts, and retryable upstream failures. It does not silently transform application-level validation errors.

For `POST /v1/transactions`, Lambda forwards the exact request body. If the first attempt fails before an authoritative application response is obtained, Lambda may retry once against the second seed using the exact same request body. It must never construct or sign a replacement transaction. If the outcome remains uncertain, the response must not claim success.

Responses include safe headers including `Cache-Control: no-store`. Logs record route, method, selected seed, latency, and status without wallet secrets, private keys, recovery phrases, signed transaction bodies, AWS credentials, or other secret material.

## AWS network and security

The Lambda execution role is least privilege and must not use `AdministratorAccess`. GitHub Actions uses the existing GitHub OIDC provider and short-lived role only; no permanent AWS credentials are introduced.

Security groups allow Lambda to reach seed private listeners on TCP 29100 and do not expose raw 28545 publicly. API Gateway is the public HTTPS boundary. Request throttling, size limits, timeouts, monitoring, and alarms are required before calling the public RPC production-ready for Testnet.

## Android behavior

The Testnet build compiles the stable API Gateway HTTPS endpoint into the application. Fresh installs use it automatically. Existing installs with a blank RPC preference migrate automatically to the compiled endpoint. Ordinary users do not see an RPC URL field. A developer-only override may remain in debug builds.

The wallet displays truthful connection states: connecting, synchronized, degraded, and offline, with current height and last successful synchronization where available. Reconnect uses bounded exponential backoff. Loss of both seeds, the AWS region, or the user's internet is reported as degraded/offline rather than as a false success.

Mainnet remains unavailable.

## Wallet transaction safety

Fee and nonce are fetched before authorization. Confirmation is shown for an immutable prepared transfer. Send state survives activity/process recreation without ambiguous retries or accidental double-send. QR payment URI parsing is behind the chain adapter and preserves a provided amount. RPC response IDs are validated and mismatches/rejections are handled explicitly.

`PortfolioState` and `CloudBackupBoundary` must be part of production flows, not test-only helpers. Cloud backup remains optional and may export only locally encrypted ciphertext.

The wallet relocks after backgrounding. Sensitive screens use `FLAG_SECURE`, with instrumentation/UI coverage for lifecycle relocking and screenshot protection. Onboarding state survives recreation without placing recovery phrases in ordinary navigation state.

## Branding

The authoritative official Sudharma source asset is the complete black-and-gold circular `SUDHARMA NETWORK / SUDH` logo supplied by the project owner. Preserve the complete logo for splash, welcome, and Settings/About. Use a carefully derived center-emblem variant for adaptive/launcher icons because ring text is unreadable at small sizes. Use a compact logo in the wallet header. Keep clear TESTNET branding throughout.

## Verification gates

Before distributing the updated APK:

- Go tests pass.
- Public-RPC unit tests pass, including route allowlist, failover, timeout, safe retry, and no-secret logging tests.
- Deployment smoke tests verify both private seeds and the public HTTPS API.
- Android unit tests pass.
- Android lint passes.
- Instrumentation/UI tests cover lifecycle relocking and `FLAG_SECURE` where the CI environment permits them.
- Secret-safety checks pass.
- The APK builds successfully and is visually inspected.
- The APK is tested on the owner's OnePlus 11R.

PR #20 remains draft/open and unmerged until all release blockers and independent review gates are resolved. The APK produced before those gates are complete must be described as a Testnet debug/intermediate build, not a final release.