# Stratum Pool Protocol Foundation Design

## Status and scope

This is the first software-only checkpoint for Sudharma Stage D. It defines an
offline, testable Stratum V1 translation core around the existing external
GPU-PoW work service. It does not start a network listener, expose a public
pool, deploy to Seed-1 or Seed-2, activate GPU-PoW, implement payouts, or claim
that Kryptex has approved or listed SUDH.

The checkpoint proves that pool sessions can translate bounded Stratum-style
messages into the existing immutable Sudharma work and submission contracts
without changing consensus semantics. Physical RTX 2060 and non-NVIDIA
OpenCL evidence remain separate gates.

## Existing contracts and constraints

- `rpc.MiningWorkService` is the only owner of active Version-2 work.
- `rpc.MiningWorkTemplate` binds the algorithm, block version, height,
  network difficulty, target, canonical header prefix and reward address to a
  domain-separated work ID.
- `rpc.MiningSolution` permits only the nonce to vary. The service rejects
  stale work, mutated templates and invalid proof.
- `compatibility/miner.Client` independently verifies a candidate before
  submitting it and tracks accepted, rejected and stale results.
- GPU-PoW activation remains disabled on public testnet and mainnet.
- Pool share accounting and payouts are operational concerns outside
  consensus. They must never mint coins or modify block rewards.
- CPU production mining fallback remains prohibited.

## Considered approaches

### A. Isolated Stratum translation core — selected

Create a transport-independent package that accepts typed session commands,
issues typed replies and notifications, maps pool jobs to immutable Sudharma
work, and validates shares before forwarding block candidates. A deterministic
transcript harness exercises the package without sockets. This gives protocol
and safety evidence while preserving the current deployment boundary.

### B. Live Stratum TCP service immediately

This would test real framing sooner, but it would also create an externally
reachable mining surface before worker identity, resource limits, nonce
partitioning and share semantics are proven. It is deferred until the offline
core is green and separately approved for deployment design.

### C. Reuse an unrelated coin's Stratum dialect

Imitating another coin could ease a single miner integration, but its job and
nonce fields would not faithfully represent Sudharma's frozen header contract.
Consensus rules must not be altered for superficial compatibility, so this
approach is rejected.

## Package boundary

Add a `pool/stratum` package containing only protocol and session logic. It
must not import `net`, open sockets, persist balances, construct blocks or
change chain state. It consumes a narrow adapter implemented around the
existing mining service:

```go
type WorkSource interface {
    CurrentWork(ctx context.Context, rewardAddress string) (Work, error)
    Submit(ctx context.Context, candidate Candidate) (SubmitResult, error)
}
```

`Work` is an immutable pool-facing snapshot of the existing Sudharma template.
`Candidate` contains the exact job ID, worker identity, assigned nonce lane and
nonce. The source `SubmitResult` distinguishes accepted block, invalid,
mutated and stale consensus outcomes. The session layer separately reports
accepted share and duplicate outcomes. It never treats an accepted low-
difficulty share as an accepted consensus block.

The first checkpoint uses a fake in-memory `WorkSource` in tests and an
adapter contract test against `rpc.MiningWorkService`. It does not wire the
adapter into `cmd/sudharma-rpcd`.

## Protocol profile

Messages use newline-delimited JSON-RPC request and response objects compatible
with the method model commonly called Stratum V1. The supported client methods
for this checkpoint are:

| Method | Purpose |
|---|---|
| `mining.subscribe` | Establish protocol capabilities and receive a session ID |
| `mining.authorize` | Bind one validated `WALLET.WORKER` identity to the session |
| `mining.submit` | Submit one nonce for the current assigned job |

The server may emit:

| Method | Purpose |
|---|---|
| `mining.set_difficulty` | Set the session share difficulty |
| `mining.notify` | Publish an immutable Sudharma job and clean-job marker |

The parser accepts request IDs that are JSON strings or integers and echoes
their exact JSON value in the response. It rejects batch requests, unknown
fields, unknown methods, duplicate object keys, non-finite numbers, malformed
UTF-8, messages above 64 KiB and trailing JSON values. One decoded line is one
message. The offline core returns typed protocol errors; it does not log wallet
or worker data.

This is a documented Sudharma Stratum profile, not a claim of exact Kryptex
wire compatibility. Exact field ordering and onboarding-specific extensions
must be validated later against Kryptex's current integration requirements.

## Identity and authorization

Authorization is syntactic in this checkpoint; it does not contact an account
database and does not accept passwords as secrets. The username must be
`WALLET.WORKER`:

- wallet is exactly 40 lowercase hexadecimal characters, matching
  `wallet.AddressFromPublicKey` output;
- worker is 1 to 32 ASCII characters from `[A-Za-z0-9_-]`;
- the full identity is at most 128 bytes;
- leading/trailing whitespace, control characters and additional dots are
  rejected; and
- an empty password or literal `x` compatibility placeholder is accepted and
  discarded; all other password content is rejected in this checkpoint.

A session must subscribe before authorizing and authorize before receiving or
submitting work. Reauthorization to a different wallet or worker fails closed.

## Job translation and nonce isolation

Each session receives an opaque 128-bit random session ID from an injected
cryptographic entropy reader and a deterministic job ID derived from the
Sudharma work ID plus a monotonically increasing generation. Tests inject a
fixed reader so transcripts remain reproducible; production construction
defaults to `crypto/rand.Reader` and fails closed on short reads or entropy
errors. The job record stores the full immutable source template;
client-supplied job fields are never trusted.

Sudharma GPU-PoW exposes one 64-bit nonce. To keep concurrent pool workers from
searching the same space, the server assigns each authorized session a unique
32-bit nonce lane for a job. The high 32 bits equal the assigned lane and the
miner searches the low 32 bits. A submission containing a nonce outside the
session lane is invalid. Lane reuse is forbidden while a job is current.

When the source work ID changes, the generation increments, all prior jobs
become stale and `mining.notify` carries `clean_jobs=true`. Reissuing an
identical work ID with different immutable fields is an internal integrity
error and no job is emitted.

## Share and block validation

Share difficulty is session-local telemetry and does not replace network
difficulty. The initial checkpoint uses a fixed positive share difficulty
supplied by trusted server configuration; variable difficulty is deferred to a
later Stage D checkpoint.

Submission processing follows this order:

1. validate session state, worker identity, job ID, nonce encoding and lane;
2. reject a previously seen `(session ID, job ID, nonce)` as duplicate;
3. reject a non-current generation as stale;
4. independently hash the exact frozen work plus nonce;
5. reject hashes above the session share target as invalid;
6. record a valid low-difficulty share as `accepted_share` without forwarding
   it to consensus;
7. when the hash also meets the immutable network target, forward the exact
   candidate once through `WorkSource.Submit`; and
8. map the source result without converting stale, invalid or mutated results
   into success.

Duplicate tracking is bounded per current job and cleared when the job becomes
stale. A configurable maximum defaults to 65,536 entries; reaching it fails
new submissions closed until clean work arrives. No unbounded worker, job or
share collection is allowed.

## Session and resource safety

The core is deterministic and safe under concurrent calls:

- maximum one authorized identity per session;
- maximum eight retained job generations per session, with older generations
  represented only by a bounded stale-ID set;
- maximum 64 KiB decoded message size;
- maximum 128-byte identity and 128-byte job ID;
- unknown methods return a protocol error without changing state;
- invalid message counts do not authorize or refresh a session;
- cancellation from the caller stops work acquisition and submission; and
- all shared state is race-tested.

Rate limiting, connection deadlines, TLS, proxy trust and IP policy belong to
the later network-listener design and are not simulated by this core.

## Offline interoperability harness

A table-driven transcript harness feeds protocol lines to a session and
records canonical replies and notifications. Frozen vectors cover:

1. subscribe, authorize, set difficulty and notify;
2. valid share below share target but above network target;
3. valid network block candidate forwarded exactly once;
4. duplicate nonce rejection;
5. stale job rejection after clean work;
6. mutation, wrong worker and wrong nonce-lane rejection;
7. malformed, oversized and unknown messages;
8. work-ID reuse with mutated immutable fields; and
9. concurrent duplicate submissions under the race detector.

The harness produces protocol evidence only. It does not represent physical
GPU validation or Kryptex acceptance.

## Test and CI requirements

- TDD red-to-green coverage for every parser, session and submission behavior.
- Fuzz tests for message decoding and identity parsing.
- Race tests for concurrent session notification and submission.
- Adapter contract tests proving immutable field preservation and exact status
  mapping to `rpc.MiningWorkService`.
- Existing GPU-PoW vectors, activation rehearsal, generic tests, vet and race
  detector remain green.
- A permanent Stage D CI gate runs the focused offline transcript suite.
- Testnet and mainnet activation constants remain disabled.

## Deferred checkpoints

The following are explicitly outside this checkpoint and require their own
design approval:

1. variable-difficulty adjustment policy and persistence;
2. a bounded TCP/TLS listener and production authentication;
3. real pool accounting, payout thresholds, fees and wallet custody;
4. Kryptex-specific extensions and official onboarding;
5. Windows/Linux/HiveOS miner configuration packages;
6. public Seed or pool deployment; and
7. any consensus activation height.

## Completion criteria

This checkpoint is complete only when the transport-independent core and
offline transcript harness are committed on the live feature branch, exact-
head GPU-PoW and generic CI pass, PR #25 documents the evidence, and no listener,
deployment, payout system, activation value or onboarding claim has been added.
