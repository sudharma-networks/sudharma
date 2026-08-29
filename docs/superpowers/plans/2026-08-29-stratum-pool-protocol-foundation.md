# Stratum Pool Protocol Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a transport-independent, bounded Stratum V1 translation core and offline transcript harness around Sudharma's immutable GPU-PoW work contract without exposing a listener or activating mining.

**Architecture:** A new `pool/stratum` package owns strict JSON message decoding, worker/session state, immutable job translation, nonce-lane assignment and share classification. It depends only on narrow work-source and verifier interfaces. A separate `rpc` adapter maps the existing `MiningWorkService` into that interface without wiring it into the node.

**Tech Stack:** Go 1.26.6 standard library, existing Sudharma `rpc`, `pow` and `wallet` packages, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-08-29-stratum-pool-protocol-foundation-design.md`

## Global Constraints

- Do not add a TCP/HTTP listener or wire Stratum into `cmd/sudharma-rpcd`.
- Do not activate GPU-PoW, change either activation sentinel, deploy to Seed-1/Seed-2, or merge PR #25.
- Do not implement payouts, balances, fees, wallet custody, variable difficulty or Kryptex-specific onboarding extensions.
- Preserve the immutable work-ID/template binding and independently classify shares before forwarding network candidates.
- Limit one message to 64 KiB, one identity to 128 bytes, one worker name to 32 ASCII characters and duplicate tracking to 65,536 current-job entries.
- Every production behavior begins with a failing test and is committed only after focused and repository tests pass.

---

### Task 1: Protocol domain types and worker identity

**Files:**
- Create: `pool/stratum/types.go`
- Create: `pool/stratum/identity.go`
- Test: `pool/stratum/identity_test.go`

**Interfaces:**
- Produces: `WorkerIdentity`, `ParseWorkerIdentity(string) (WorkerIdentity, error)`, `Work`, `Candidate`, `SourceResult`, `WorkSource`, `ShareVerifier`, `SubmitStatus`.
- Consumes: only `context.Context` and standard-library value types.

- [ ] **Step 1: Write the failing worker-identity tests**

Cover a 40-character lowercase hexadecimal wallet plus `.rig_01`, and reject uppercase wallet hex, wrong wallet length, empty/33-byte workers, additional dots, whitespace, controls and characters outside `[A-Za-z0-9_-]`. Assert parsed wallet and worker separately.

```go
func TestParseWorkerIdentity(t *testing.T) {
    got, err := ParseWorkerIdentity("9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01")
    if err != nil { t.Fatal(err) }
    if got.Wallet != "9ccdc094489874bed888ffe4bdf9b8298f4c5131" || got.Worker != "rig_01" {
        t.Fatalf("unexpected identity: %+v", got)
    }
}
```

- [ ] **Step 2: Run the test and verify RED**

Run: `go test ./pool/stratum -run 'TestParseWorkerIdentity' -count=1`

Expected: FAIL because `ParseWorkerIdentity` and its types do not exist.

- [ ] **Step 3: Add minimal domain types and identity parsing**

Use these stable shapes:

```go
type WorkerIdentity struct { Wallet, Worker string }
type Work struct {
    WorkID, Algorithm, TargetHex, HeaderPrefixHex, RewardAddress string
    Version uint32
    Height uint64
    Difficulty uint32
}
type Candidate struct { Work Work; JobID string; Identity WorkerIdentity; Lane uint32; Nonce uint64 }
type SourceResult string
const (
    SourceAccepted SourceResult = "accepted"
    SourceInvalid SourceResult = "invalid"
    SourceStale SourceResult = "stale"
    SourceMutated SourceResult = "mutated"
)
type WorkSource interface {
    CurrentWork(context.Context, string) (Work, error)
    Submit(context.Context, Candidate) (SourceResult, error)
}
type ShareVerifier interface {
    MeetsTarget(context.Context, Work, uint64, [32]byte) (bool, error)
}
```

Parse exactly one dot, exactly 40 lowercase hexadecimal wallet bytes and a 1–32 byte ASCII worker.

- [ ] **Step 4: Verify GREEN**

Run: `gofmt -w pool/stratum/*.go && go test ./pool/stratum -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

Commit: `feat(pool): define Stratum worker and work contracts`

### Task 2: Strict bounded Stratum message codec

**Files:**
- Create: `pool/stratum/codec.go`
- Test: `pool/stratum/codec_test.go`
- Test: `pool/stratum/codec_fuzz_test.go`

**Interfaces:**
- Produces: `Request`, `Response`, `Notification`, `ProtocolError`, `DecodeRequest([]byte) (Request, error)`, `EncodeMessage(any) ([]byte, error)`.
- Consumes: the 64 KiB limit from `types.go`.

- [ ] **Step 1: Write failing codec tests**

Use table cases for string/integer IDs, `mining.subscribe`, `mining.authorize` and `mining.submit`. Reject empty input, malformed UTF-8, arrays/batches, duplicate keys, unknown top-level fields, fractional/null/boolean IDs, unknown methods, trailing JSON, and 65,537-byte lines. Verify encoded messages end in one newline.

- [ ] **Step 2: Verify RED**

Run: `go test ./pool/stratum -run 'TestDecodeRequest|TestEncodeMessage' -count=1`

Expected: FAIL because the codec is missing.

- [ ] **Step 3: Implement strict decoding**

First validate UTF-8 and byte length. Walk `json.Decoder.Token` over the complete value to reject duplicate keys at every object depth. Then decode into:

```go
type Request struct {
    ID json.RawMessage `json:"id"`
    Method string `json:"method"`
    Params json.RawMessage `json:"params"`
}
```

Use `DisallowUnknownFields`, `UseNumber`, an EOF check, exact method allow-listing, and explicit ID validation as either a JSON string or base-10 integer. Return stable JSON-RPC error codes for parse error, invalid request, method not found and invalid params.

- [ ] **Step 4: Add fuzz seeds and verify GREEN**

Seed the three supported messages plus duplicate-key, oversized and malformed examples. The fuzz target must never panic and must never return a successful request for invalid UTF-8 or multiple JSON values.

Run: `gofmt -w pool/stratum/*.go && go test ./pool/stratum -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

Commit: `feat(pool): decode bounded Stratum messages strictly`

### Task 3: Subscription and authorization session state

**Files:**
- Create: `pool/stratum/session.go`
- Test: `pool/stratum/session_test.go`

**Interfaces:**
- Produces: `NewSession(io.Reader, WorkSource, ShareVerifier, Config) (*Session, error)`, `Session.Handle(context.Context, []byte) ([]Message, error)`, `SessionID()`, and `LaneSource` with `Acquire(workID, sessionID string) (uint32, error)` plus `Release(workID, sessionID string)`.
- Consumes: identity and codec APIs from Tasks 1–2.

- [ ] **Step 1: Write failing lifecycle tests**

Prove authorization fails before subscribe, empty or `x` passwords are discarded, other passwords fail, repeat authorization to the same identity is idempotent, different reauthorization fails, and an entropy short read prevents session construction. Use a fixed 16-byte reader and assert the 32-character lowercase hexadecimal session ID.

- [ ] **Step 2: Verify RED**

Run: `go test ./pool/stratum -run 'TestSession.*Subscribe|TestSession.*Authorize|TestNewSession' -count=1`

Expected: FAIL because `Session` is missing.

- [ ] **Step 3: Implement the minimal state machine**

`Config` contains `ShareDifficulty uint32`, `MaxDuplicateShares int`, an optional entropy reader and a required `LaneSource`. Default entropy is `crypto/rand.Reader`; default duplicate limit is 65,536. Lifecycle tests use a fixed lane-source fixture even though no job is acquired yet. Protect state with one mutex. `mining.subscribe` returns the session ID. `mining.authorize` validates exactly `[username,password]` and returns `true` only after subscription.

- [ ] **Step 4: Verify GREEN and race safety**

Run: `gofmt -w pool/stratum/*.go && go test -race ./pool/stratum -run 'TestSession|TestNewSession' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

Commit: `feat(pool): enforce Stratum session authorization`

### Task 4: Immutable jobs and nonce-lane assignment

**Files:**
- Create: `pool/stratum/job.go`
- Test: `pool/stratum/job_test.go`
- Modify: `pool/stratum/session.go`

**Interfaces:**
- Produces: `Session.RefreshWork(context.Context) ([]Message, error)`, `LaneAllocator`, `NewLaneAllocator() *LaneAllocator`, and immutable internal `job` snapshots.
- Consumes: `WorkSource.CurrentWork`, authorized identity and `Work`.

- [ ] **Step 1: Write failing job tests**

Assert that authorized refresh emits `mining.set_difficulty` then `mining.notify`; notification binds job ID, algorithm, height, targets, header prefix, lane and `clean_jobs=true`. Identical work yields no replacement notification. A changed work ID increments generation and stales the old job. Reusing one work ID with any changed immutable field fails closed.

- [ ] **Step 2: Verify RED**

Run: `go test ./pool/stratum -run 'TestSessionRefreshWork' -count=1`

Expected: FAIL because job refresh is missing.

- [ ] **Step 3: Implement job derivation and lane allocation**

Derive job ID as SHA-256 over `SUDHARMA-STRATUM-JOB-V1\x00`, source work ID, session ID and big-endian generation. Add a mutex-protected `LaneAllocator` shared through `Config`; allocate the first unused 32-bit lane beginning at the first four bytes of SHA-256 over `SUDHARMA-STRATUM-LANE-V1\x00`, source work ID and session ID, probing upward on collision. Release a session's lane when that work ID becomes stale. `NewSession` rejects a nil allocator so every group of sessions must deliberately share one coordinator. Store at most the current job plus eight bounded stale job IDs.

- [ ] **Step 4: Verify GREEN**

Run: `gofmt -w pool/stratum/*.go && go test -race ./pool/stratum -run 'TestSessionRefreshWork' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

Commit: `feat(pool): translate immutable work into Stratum jobs`

### Task 5: Share, duplicate, stale and block-candidate classification

**Files:**
- Create: `pool/stratum/submit.go`
- Test: `pool/stratum/submit_test.go`
- Modify: `pool/stratum/session.go`

**Interfaces:**
- Produces: `mining.submit` handling with `accepted_share`, `accepted_block`, `invalid`, `duplicate`, `stale` and `mutated` outcomes.
- Consumes: `ShareVerifier.MeetsTarget`, `WorkSource.Submit`, active job and assigned nonce lane.

- [ ] **Step 1: Write failing submission tests**

Cover authorization requirement, exact worker/job binding, unsigned 64-bit hexadecimal nonce decoding, high-32-bit lane enforcement, invalid share, accepted low-difficulty share without source submission, network candidate submitted exactly once, exact source-status mapping, duplicate nonce, stale generation, 65,536-entry bound and concurrent same-nonce submission yielding one winner.

- [ ] **Step 2: Verify RED**

Run: `go test ./pool/stratum -run 'TestSessionSubmit' -count=1`

Expected: FAIL because submission handling is missing.

- [ ] **Step 3: Implement share classification**

Parse params exactly as `[worker,job_id,nonce_hex]`. Convert configured share difficulty with the same 256-bit target formula used by `pow.TargetFromDifficulty`. Decode the immutable network target from `Work.TargetHex`. Call the verifier first for share target, then network target. Record the duplicate key atomically before forwarding; remove it only when verification returns an operational error, not for a rejected share. Forward only network-target candidates.

- [ ] **Step 4: Verify GREEN under race**

Run: `gofmt -w pool/stratum/*.go && go test -race ./pool/stratum -run 'TestSessionSubmit' -count=1`

Expected: PASS with exactly one forwarded concurrent candidate.

- [ ] **Step 5: Commit**

Commit: `feat(pool): reject duplicate and stale Stratum shares`

### Task 6: RPC mining-service adapter

**Files:**
- Create: `rpc/stratum_adapter.go`
- Test: `rpc/stratum_adapter_test.go`

**Interfaces:**
- Produces: `NewStratumWorkSource(*MiningWorkService, MiningBlockProvider) (*StratumWorkSource, error)` implementing `stratum.WorkSource`.
- Consumes: `MiningWorkService.Issue`, `MiningWorkService.Submit`, `MiningBlockProvider`, `stratum.Work`, `stratum.Candidate`.

- [ ] **Step 1: Write failing adapter contract tests**

Assert nil dependencies fail, reward address is placed into a copied candidate block before issue, every immutable template field maps exactly, and accepted/invalid/stale/mutated statuses map one-for-one. Assert candidate job/identity metadata cannot mutate the RPC solution.

- [ ] **Step 2: Verify RED**

Run: `go test ./rpc -run 'TestStratumWorkSource' -count=1`

Expected: FAIL because the adapter does not exist.

- [ ] **Step 3: Implement the adapter without node wiring**

The adapter calls the provider, copies the block, sets only `MinerAddress` from the validated reward address, issues through the service, and stores the exact returned template by work ID in a mutex-protected bounded current snapshot. Submission reconstructs `MiningSolution` exclusively from that stored template plus candidate nonce and calls `MiningWorkService.Submit`.

- [ ] **Step 4: Verify GREEN and dependency direction**

Run:

```bash
gofmt -w rpc/stratum_adapter*.go
go test -race ./rpc ./pool/stratum -count=1
if go list -deps ./pool/stratum | grep -q '^github.com/sudharma-networks/sudharma/rpc$'; then
  echo 'pool/stratum must not import rpc'
  exit 1
fi
```

Expected: tests PASS and `pool/stratum` does not depend on `rpc`.

- [ ] **Step 5: Commit**

Commit: `feat(pool): adapt immutable RPC mining work to Stratum`

### Task 7: Frozen transcript harness and permanent CI gate

**Files:**
- Create: `pool/stratum/transcript_test.go`
- Create: `docs/stratum/SUDHARMA_STRATUM_V1.md`
- Modify: `.github/workflows/gpu-pow-v1-ci.yml`
- Modify: `docs/superpowers/plans/2026-08-29-stratum-pool-protocol-foundation.md`

**Interfaces:**
- Produces: canonical offline transcript evidence and `Stage D offline Stratum protocol gate` workflow step.
- Consumes: all Tasks 1–6 APIs.

- [ ] **Step 1: Write the failing transcript test**

Freeze exact request/reply lines for subscribe, authorize, difficulty, notify, accepted share, accepted block, duplicate, stale, wrong lane and malformed request. Use fixed entropy, fixed work and deterministic verifier/source fixtures. Compare the full newline-delimited transcript byte-for-byte.

- [ ] **Step 2: Verify RED**

Run: `go test ./pool/stratum -run TestOfflineStratumTranscript -count=1 -v`

Expected: FAIL until the frozen expected transcript matches the complete implementation.

- [ ] **Step 3: Finalize transcript and operator-facing protocol documentation**

Document the exact supported methods, 64-bit nonce/lane format, identity rules, result meanings, limits and explicit exclusions. State that the profile is not a public endpoint, payout pool or Kryptex onboarding claim.

- [ ] **Step 4: Add the permanent workflow gate**

Add after the activation rehearsal:

```yaml
      - name: Stage D offline Stratum protocol gate
        run: go test ./pool/stratum ./rpc -run 'Stratum|OfflineStratumTranscript' -count=1 -v
```

- [ ] **Step 5: Run complete local verification**

Run:

```bash
gofmt -w pool/stratum/*.go rpc/stratum_adapter*.go
go vet ./...
go test ./... -count=1
go test -race ./pool/stratum ./rpc -count=1
go test ./pow -run TestGPUV1NetworkActivationDefaultsRemainDisabled -count=1
git diff --check
```

Expected: all commands PASS and both activation defaults remain disabled.

- [ ] **Step 6: Mark completed plan checkboxes and commit**

Commit: `test(pool): freeze offline Stratum interoperability gate`

- [ ] **Step 7: Push and verify exact live GitHub head**

Fast-forward `feature/gpu-pow-v1`, monitor GPU-PoW v1 and generic CI through completion, fix any failure test-first, and verify PR #25 remains draft/open/unmerged.

- [ ] **Step 8: Update PR #25 checkpoint**

Record the exact commit and workflow run numbers, the offline-only boundary, and remaining physical/Kryptex/listener/payout gates. Do not claim Stage D complete until both exact-head workflows pass.
