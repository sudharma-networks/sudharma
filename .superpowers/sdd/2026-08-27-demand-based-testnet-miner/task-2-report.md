# Task 2 report: deterministic demand supervisor

## Summary

Added a deterministic, single-goroutine `demandminer.Supervisor` that reads
only aggregate status, validates the configured public-testnet identity before
consulting the mempool count, and invokes the injected bounded miner only when
that count is positive. It has no transaction, wallet, signing, or priority
interfaces.

The loop uses `PollEvery` for an empty mempool, `Cooldown` after a successful
block, and `FailureBackoff` after status, malformed-count, or miner failures.
Identity mismatches and cancellation return immediately. The injected logger
receives only recoverable error event names and error text.

## Files

- `demandminer/supervisor.go`
- `demandminer/supervisor_test.go`

## TDD evidence and tooling blocker

Tests were written before `supervisor.go`. They cover empty and positive
mempools, the terminal wrong-identity path, status and miner failure backoff,
cooldown before a second positive-mempool block, cancellation without another
poll or miner invocation, malformed negative counts, and the one-active-miner
invariant.

The required RED and GREEN commands could not execute: this environment has
no `go` executable and also lacks `gofmt`. Consequently, no test or race-pass
result is claimed.

## Commands and results

- `go version` — blocked: `/bin/bash: go: command not found`.
- `go test ./demandminer -run 'TestSupervisor' -count=1` — blocked before
  test discovery by the missing `go` executable.
- `go test ./demandminer -run 'TestSupervisorNegativeMempoolUsesFailureBackoffWithoutMining' -count=1`
  — blocked before test discovery by the missing `go` executable.
- `go test ./demandminer -run 'TestSupervisorCancellationAfterStatusDoesNotStartMiner' -count=1`
  — blocked before test discovery by the missing `go` executable.
- `go test -race ./demandminer -run 'TestSupervisor' -count=1` — blocked
  before test discovery by the missing `go` executable.
- Checked standard Go locations plus the provided runtime root for `go` and
  `gofmt` — neither executable was present.
- `git diff --cached --check` — passed before commit.
- `bash scripts/check-tracked-secrets.sh` — passed: `PASS: no prohibited
  secret-like files are tracked`.
- `git show --check HEAD` — passed after commit.

## Self-review

- Every successful status has its network, coin, and symbol compared exactly
  with Task 1's validated expected identities before the `Mempool` field is
  examined.
- `MineOne` is called synchronously in the sole loop, so the supervisor cannot
  overlap native miner calls.
- Empty, invalid-negative, status-error, miner-error, success, and
  cancellation branches all exit or sleep with the specified interval.
- The package accepts only status, mining, sleep, and logging ports; it cannot
  create, sign, inspect, or prioritize transactions.
- The public API has dependency checks so a missing source, miner, or sleeper
  fails deterministically instead of panicking.

## Concerns

Go compilation, `gofmt`, focused tests, and the race detector remain blocked
by the missing Go toolchain. A Go-enabled CI or development environment must
run `go test -race ./demandminer -run 'TestSupervisor' -count=1` and format the
two new Go files before integration.

## Fix Round 1

### Important 1: immutable, fixed testnet identity

Root cause: `Run` previously read the exported `Supervisor.Config` identity
fields directly. A caller could mutate those values after construction and
make a matching non-testnet status reach `MineOne`.

`NewSupervisor` now validates the provided configuration at its construction
boundary, captures the validation error, and stores a private `config`
snapshot used for all duration decisions. `Run` returns that construction-time
validation error before polling. Status validation now compares only the fixed
public-testnet values `sudharma`, `Sudharma`, and `SUDH`; the exported `Config`
field is retained only for inspection and is not used by the loop.

`TestSupervisorRejectsUnvalidatedMainnetConfigBeforePolling` proves that an
invalid mainnet identity is rejected before any status request or mining.
`TestSupervisorIgnoresMutatedPublicConfigIdentity` proves that mutating the
public field after construction cannot make a mainnet status reach `MineOne`.

### Important 2: cooldown ordering test

Root cause: the previous fake sleeper returned immediately, so checking the
final duration list did not prove that the second miner call waited for the
first cooldown.

`TestSupervisorMinesRemainingWorkOnlyAfterCooldown` now uses a blocking,
context-aware sleeper. It waits for the first cooldown to begin, asserts that
only one `MineOne` call has happened, then releases the cooldown and waits for
the second call.

### Important 3: bounded test waits

The active-miner test now uses bounded `select` waits with one-second test
deadlines for miner-start and supervisor-completion signals. The fake miner's
optional block also observes context cancellation, so a failed assertion does
not leave that test goroutine permanently blocked.

### Minor review items

`TestSupervisorWrongIdentityFailsClosed` is now a table test covering wrong
network, coin, and symbol. The unvalidated mainnet configuration rejection is
covered separately before polling. All review regression coverage is in
`demandminer/supervisor_test.go`.

### Fix-round commands and results

- `go test ./demandminer -run 'TestSupervisor' -count=1` — blocked before
  test discovery: `/bin/bash: go: command not found`.
- `go test -race ./demandminer -run 'TestSupervisor' -count=1` — blocked
  before test discovery: `/bin/bash: go: command not found`.
- `command -v go` and `command -v gofmt` — no executable found.
- `git diff --cached --check` — passed.
- `bash scripts/check-tracked-secrets.sh` — passed: `PASS: no prohibited
  secret-like files are tracked`.

CI must run `gofmt` and the focused race command above in a Go-enabled
environment; they cannot be run or claimed in this environment.
