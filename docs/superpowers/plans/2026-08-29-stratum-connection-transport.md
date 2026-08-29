# Bounded Stratum Connection Transport Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an offline-only, injected-`net.Conn` transport that safely carries the frozen Stage D Sudharma Stratum profile without opening a listener or activating/deploying GPU-PoW.

**Architecture:** New subpackage `pool/stratum/transport` owns bounded newline framing, one-connection request lifecycle, serialized writes, per-connection abuse controls, periodic work refresh, I/O deadlines and cancellation. It imports the completed `pool/stratum` core and never owns consensus, RPC templates, sockets/listeners, TLS configuration, payout state or Kryptex-specific behavior.

**Tech Stack:** Go 1.26.6 standard library (`bufio`, `context`, `errors`, `io`, `net`, `sync`, `time`, `go/ast`, `go/parser`, `go/token`), existing `pool/stratum`, existing `rpc`, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-08-29-stratum-connection-transport-design.md`

## Global Constraints

- Do not call `net.Listen`, `tls.Listen`, `http.Serve`, `http.ListenAndServe`, or create any accept loop in Stage E.
- Do not wire transport into `cmd/sudharma-rpcd` or any Seed-1/Seed-2 service.
- Do not add TLS certificate loading, proxy trust, IP policy, vardiff, payouts, balances, fees, custody, persistent accounts or Kryptex-specific extensions.
- Do not set a finite GPU-PoW activation height, change disabled activation defaults, enable unrestricted GPU mining, deploy consensus, merge PR #25 or activate mainnet.
- Preserve the Stage D 64 KiB message bound, stable protocol errors, `WALLET.WORKER`, immutable job/work binding, nonce lanes and share classification unchanged.
- `pool/stratum/transport` may import `pool/stratum`; `pool/stratum` must not import `pool/stratum/transport`; `rpc` remains outside both layers.
- `ServeConn` owns and closes exactly one injected `net.Conn`; no goroutine started by it may remain after it returns.
- Every behavior change starts with a failing test and receives exact-head GitHub CI verification before the task is marked complete.

---

### Task 1: Configuration and bounded request-line framing

**Files:**
- Create: `pool/stratum/transport/config.go`
- Create: `pool/stratum/transport/config_test.go`
- Create: `pool/stratum/transport/line.go`
- Create: `pool/stratum/transport/line_test.go`

**Interfaces:**
- Consumes: `*stratum.Session` from `github.com/sudharma-networks/sudharma/pool/stratum`.
- Produces: public `SessionFactory`, public `Config`, stable sentinel errors, internal `normalizedConfig`, `normalizeConfig(Config)`, `newRequestReader(io.Reader)`, and `readRequestLine(*bufio.Reader)`.

Public/API constants and types:

```go
type SessionFactory func() (*stratum.Session, error)

type Config struct {
    ReadTimeout       time.Duration
    WriteTimeout      time.Duration
    RefreshInterval   time.Duration
    MaxProtocolErrors int
    RequestsPerSecond uint32
    Burst             uint32
}

var (
    ErrInvalidConfig  = errors.New("invalid Stratum transport configuration")
    ErrLineTooLong    = errors.New("Stratum request line too long")
    ErrProtocolBudget = errors.New("Stratum protocol error budget exceeded")
    ErrRateLimited    = errors.New("Stratum connection rate limit exceeded")
)
```

Internal defaults are exactly 30s read timeout, 10s write timeout, 5s refresh interval, 8 recoverable protocol errors, 20 requests/second and burst 40. `maxRequestBytes` is exactly `64 * 1024` and the buffered reader capacity is `maxRequestBytes + 2` so both a maximum-sized LF line and maximum-sized CRLF line fit without unbounded allocation.

- [ ] **Step 1: Write failing configuration tests**

Add table tests proving zero values resolve to all six defaults and each negative duration is rejected with `errors.Is(err, ErrInvalidConfig)`.

```go
func TestNormalizeConfigDefaults(t *testing.T) {
    got, err := normalizeConfig(Config{})
    if err != nil { t.Fatal(err) }
    if got.readTimeout != 30*time.Second || got.writeTimeout != 10*time.Second || got.refreshInterval != 5*time.Second {
        t.Fatalf("unexpected time defaults: %+v", got)
    }
    if got.maxProtocolErrors != 8 || got.requestsPerSecond != 20 || got.burst != 40 {
        t.Fatalf("unexpected abuse defaults: %+v", got)
    }
}
```

- [ ] **Step 2: Run the configuration tests and verify RED**

Run: `go test ./pool/stratum/transport -run 'TestNormalizeConfig' -count=1`

Expected: FAIL because the transport package/configuration types do not exist.

- [ ] **Step 3: Implement minimal configuration normalization**

Create `normalizedConfig` with resolved non-zero fields. Treat zero as default and reject only negative duration values. Do not add deployment-specific configuration.

- [ ] **Step 4: Write failing bounded-line tests**

Cover LF, CRLF, a non-empty unterminated final line at EOF, clean empty EOF, exactly 65,536 content bytes, 65,537 content bytes, and an overlong fragmented reader that never presents a delimiter before the fixed buffer fills.

```go
func TestReadRequestLineRejectsOverLimit(t *testing.T) {
    reader := newRequestReader(strings.NewReader(strings.Repeat("x", 64*1024+1) + "\n"))
    if _, err := readRequestLine(reader); !errors.Is(err, ErrLineTooLong) {
        t.Fatalf("error = %v, want ErrLineTooLong", err)
    }
}
```

- [ ] **Step 5: Run line tests and verify RED**

Run: `go test ./pool/stratum/transport -run 'TestReadRequestLine' -count=1`

Expected: FAIL because the bounded line reader is missing.

- [ ] **Step 6: Implement bounded framing**

Use `bufio.NewReaderSize(r, maxRequestBytes+2)` and `ReadSlice('\n')`. `bufio.ErrBufferFull` maps to `ErrLineTooLong`; trim one LF and then one optional CR; reject post-trim content above 65,536 bytes; return a bounded non-empty EOF fragment as a normal line; return `io.EOF` only for empty EOF. Copy the returned line before the next reader operation.

- [ ] **Step 7: Verify Task 1 GREEN**

Run:

```bash
gofmt -w pool/stratum/transport/*.go
go test -race ./pool/stratum/transport -count=1
go vet ./pool/stratum/transport
```

Expected: PASS with no formatting output.

- [ ] **Step 8: Commit and exact-head verify**

Commit: `feat(pool): bound Stratum connection framing`

Then verify both GPU-PoW v1 CI and generic CI succeed on that exact commit before Task 2.

---

### Task 2: Single injected connection lifecycle and immediate work delivery

**Files:**
- Create: `pool/stratum/transport/writer.go`
- Create: `pool/stratum/transport/serve.go`
- Create: `pool/stratum/transport/serve_test.go`

**Interfaces:**
- Consumes: `SessionFactory`, normalized `Config`, bounded request reader, `stratum.DecodeRequest`, `(*stratum.Session).Handle`, `(*stratum.Session).RefreshWork`, `stratum.EncodeMessage`.
- Produces: public `ServeConn(context.Context, net.Conn, SessionFactory, Config) error`, internal mutex-protected `messageWriter`, protocol-error framing, and authorization-success detection.

Task 2 intentionally has no periodic refresh goroutine yet. It proves the base stream/request lifecycle and immediate work delivery after authorization; Task 4 adds periodic refresh and I/O deadlines test-first.

- [ ] **Step 1: Write failing `net.Pipe` lifecycle tests**

Build real Stage D sessions from a fixed source/verifier/lane fixture. Prove:

1. one factory call creates one session;
2. a subscribe request split across multiple client writes is reconstructed correctly;
3. subscribe + authorize coalesced into one client write are processed as two requests;
4. CRLF input is accepted;
5. successful authorize immediately yields `mining.set_difficulty` followed by `mining.notify`;
6. client EOF after valid traffic causes `ServeConn` to return nil and close its owned endpoint.

The test must decode every server line as JSON rather than matching partial writes.

```go
server, client := net.Pipe()
done := make(chan error, 1)
go func() { done <- ServeConn(context.Background(), server, factory, Config{}) }()
// Write fragmented/coalesced requests on client while concurrently reading complete response lines.
```

- [ ] **Step 2: Run lifecycle tests and verify RED**

Run: `go test ./pool/stratum/transport -run 'TestServeConn.*Lifecycle|TestServeConn.*Fragment|TestServeConn.*CRLF' -count=1 -v`

Expected: FAIL because `ServeConn` and the connection writer do not exist.

- [ ] **Step 3: Implement serialized message writing**

`messageWriter` owns one `net.Conn` plus one `sync.Mutex`. `WriteMessages` first encodes all `stratum.Message` values into one local byte buffer using `stratum.EncodeMessage`, then holds the mutex while writing the complete buffer with a `writeAll` loop. Task 2 does not set write deadlines yet.

- [ ] **Step 4: Implement base `ServeConn` request loop**

Validate non-nil connection/factory and normalized config, defer `conn.Close()`, construct exactly one session, and read bounded lines until clean EOF. Before `Session.Handle`, call `stratum.DecodeRequest` only to capture the method for transport lifecycle decisions; then call `Session.Handle` exactly once. A returned `*stratum.ProtocolError` is framed as:

```go
stratum.Response{
    ID:     json.RawMessage("null"),
    Result: nil,
    Error:  protocolErr,
}
```

Non-protocol session errors terminate with operation-only wrapping. Successful session messages are written through `messageWriter`.

After a successful `mining.authorize` response whose result is boolean `true`, call `session.RefreshWork(ctx)` immediately and write its messages through the same writer. Do not start a background refresh loop in this task.

- [ ] **Step 5: Verify lifecycle GREEN under race**

Run:

```bash
gofmt -w pool/stratum/transport/*.go
go test -race ./pool/stratum/transport -run 'TestServeConn' -count=1 -v
go test ./pool/stratum -run TestOfflineStratumTranscript -count=1 -v
```

Expected: transport lifecycle tests PASS and the frozen Stage D transcript remains unchanged.

- [ ] **Step 6: Commit and exact-head verify**

Commit: `feat(pool): serve one injected Stratum connection`

Then require exact-head GPU-PoW v1 CI and generic CI PASS before Task 3.

---

### Task 3: Protocol-error budget and per-connection token bucket

**Files:**
- Create: `pool/stratum/transport/limiter.go`
- Create: `pool/stratum/transport/limiter_test.go`
- Create: `pool/stratum/transport/abuse_test.go`
- Modify: `pool/stratum/transport/serve.go`

**Interfaces:**
- Consumes: resolved `requestsPerSecond`, `burst`, `maxProtocolErrors`, complete request boundaries from Task 1, Task 2 `ServeConn` loop.
- Produces: internal `tokenBucket`, `newTokenBucket(rate, burst uint32, now time.Time)`, `(*tokenBucket).Allow(now time.Time) bool`, terminal `ErrRateLimited` and terminal `ErrProtocolBudget` enforcement.

- [ ] **Step 1: Write failing deterministic token-bucket tests**

Use explicit timestamps, never sleeps. Prove initial capacity equals burst, one request consumes one token, exhaustion rejects, fractional elapsed time replenishes according to `RequestsPerSecond`, refill caps at burst, and a backward timestamp does not mint tokens.

```go
start := time.Unix(1000, 0)
b := newTokenBucket(2, 2, start)
if !b.Allow(start) || !b.Allow(start) || b.Allow(start) {
    t.Fatal("unexpected initial burst behavior")
}
if !b.Allow(start.Add(500 * time.Millisecond)) {
    t.Fatal("one token should refill after 500ms at 2 requests/sec")
}
```

- [ ] **Step 2: Run limiter tests and verify RED**

Run: `go test ./pool/stratum/transport -run 'TestTokenBucket' -count=1`

Expected: FAIL because the token bucket does not exist.

- [ ] **Step 3: Implement token bucket**

Track `tokens float64`, `rate float64`, `capacity float64`, and `last time.Time`. On `Allow(now)`, refill only when `now.After(last)`, cap at capacity, advance `last`, then consume exactly one token when available.

- [ ] **Step 4: Write failing integration tests for rate and protocol budgets**

Use `net.Pipe` and a factory counter. With rate 1/burst 1, send two complete requests in one client write and assert the first is handled while the second causes `errors.Is(serveErr, ErrRateLimited)`. Verify the rate-limited request produces no Stage D response and does not create another session.

With `MaxProtocolErrors: 2`, send three bounded malformed requests. Assert all three receive stable `-32700` responses, the first two keep the connection alive, and after framing the third `ServeConn` returns `ErrProtocolBudget`.

Add an oversized-line integration test proving 65,537 bytes cannot bypass the limit by fragmented client writes; it receives one best-effort `-32600` response and terminates with `ErrLineTooLong`.

- [ ] **Step 5: Run abuse tests and verify RED**

Run: `go test ./pool/stratum/transport -run 'TestServeConn.*Rate|TestServeConn.*ProtocolBudget|TestServeConn.*Oversized' -count=1 -v`

Expected: FAIL because Task 2 has no abuse enforcement.

- [ ] **Step 6: Integrate abuse controls**

Initialize one token bucket per connection when `ServeConn` starts. Consume one token after a complete bounded line is assembled and before any Stage D decode/handle call. On exhaustion return `ErrRateLimited` without calling `Session.Handle` for that line.

Count only returned `*stratum.ProtocolError` values against the protocol budget. Frame each protocol error first, increment the count, and return `ErrProtocolBudget` when `count > maxProtocolErrors` so exactly the configured number remain recoverable.

When `readRequestLine` returns `ErrLineTooLong`, best-effort write an `id:null` response containing `ProtocolError{Code: -32600, Message: "invalid request"}`, then return `ErrLineTooLong` regardless of the protocol-error count.

- [ ] **Step 7: Verify Task 3 GREEN**

Run:

```bash
gofmt -w pool/stratum/transport/*.go
go test -race ./pool/stratum/transport -count=1
go test ./pool/stratum -run TestOfflineStratumTranscript -count=1 -v
go vet ./pool/stratum/transport
```

Expected: all PASS and Stage D transcript unchanged.

- [ ] **Step 8: Commit and exact-head verify**

Commit: `feat(pool): bound Stratum connection abuse`

Require exact-head GPU-PoW v1 CI and generic CI PASS before Task 4.

---

### Task 4: Periodic immutable-work refresh, deadlines and cancellation

**Files:**
- Create: `pool/stratum/transport/refresh.go`
- Create: `pool/stratum/transport/refresh_test.go`
- Create: `pool/stratum/transport/deadline_test.go`
- Modify: `pool/stratum/transport/writer.go`
- Modify: `pool/stratum/transport/serve.go`

**Interfaces:**
- Consumes: Task 2 `messageWriter`, successful authorization detection, resolved read/write/refresh durations, `Session.RefreshWork`.
- Produces: one periodic refresh pump per authorized connection, internal ticker factory used only by same-package tests, write/read deadlines, cancellation wakeup, and deterministic refresh-error propagation.

Use these internal timing interfaces so no test clock leaks into the public API:

```go
type ticker interface {
    C() <-chan time.Time
    Stop()
}

type tickerFactory func(time.Duration) ticker
```

`ServeConn` calls an internal `serveConn(..., newTicker tickerFactory)` using a real `time.NewTicker` wrapper. Same-package tests call the internal function with a manually triggered ticker.

- [ ] **Step 1: Write failing refresh-pump tests**

Using a manual ticker and mutable thread-safe Stage D source, prove:

1. authorization performs the immediate refresh from Task 2 and starts exactly one ticker;
2. repeat authorization to the same identity does not start a second ticker;
3. a tick with identical work emits nothing;
4. after changing the immutable source work ID, one tick emits exactly one difficulty/notify pair;
5. a source refresh error becomes the error returned by `ServeConn` rather than a secondary read error;
6. simultaneous submit-response traffic and refresh notifications decode as complete, non-interleaved JSON lines.

- [ ] **Step 2: Run refresh tests and verify RED**

Run: `go test ./pool/stratum/transport -run 'TestServeConn.*Refresh|TestServeConn.*Serialized' -count=1 -v`

Expected: FAIL because periodic refresh/ticker coordination is missing.

- [ ] **Step 3: Implement one refresh pump and terminal-error channel**

Start the pump only after the first successful authorize + immediate refresh. The goroutine owns one ticker, calls `Session.RefreshWork` on each tick and writes through the same `messageWriter`.

Use a buffered one-element terminal-error channel. On the first refresh or refresh-delivery failure, send one operation-wrapped error and call `conn.SetReadDeadline(time.Now())` so a blocked request read wakes. The request loop checks that channel before interpreting the resulting read error and returns the refresh error. A derived context stops the pump; `ServeConn` joins the pump before return.

- [ ] **Step 4: Write failing deadline/cancellation tests**

Use `net.Pipe` plus short, test-only explicit durations. Prove:

- no client input causes a read timeout returned with `read Stratum request` context;
- a client that sends subscribe but never reads causes the server response write to time out with `write Stratum response` context;
- canceling the parent context while blocked on read causes `ServeConn` to return `context.Canceled` promptly;
- after each path, the server side is closed and the `ServeConn` goroutine has returned.

Use channels/select timeouts only as test deadlock guards, not as protocol timing assertions.

- [ ] **Step 5: Run deadline tests and verify RED**

Run: `go test ./pool/stratum/transport -run 'TestServeConn.*Deadline|TestServeConn.*Cancel' -count=1 -v`

Expected: FAIL because Task 3 does not set I/O deadlines or install a cancellation watcher.

- [ ] **Step 6: Implement I/O deadlines and cancellation watcher**

Before every blocking request read call `conn.SetReadDeadline(time.Now().Add(readTimeout))`. In `messageWriter.WriteMessages`, while holding the writer mutex and before writing the batch, call `SetWriteDeadline(time.Now().Add(writeTimeout))`.

Create a derived context and one cancellation watcher. When it observes cancellation it calls `conn.SetDeadline(time.Now())` to wake blocked I/O. On any request-loop terminal path cancel the derived context, stop/join refresh, stop/join the watcher, close the owned connection, and prefer `ctx.Err()` when cancellation caused the wakeup.

- [ ] **Step 7: Verify Task 4 GREEN and race safety**

Run:

```bash
gofmt -w pool/stratum/transport/*.go
go test -race ./pool/stratum/transport -count=1
go test -race ./pool/stratum/... ./rpc -count=1
go vet ./pool/stratum/... ./rpc
```

Expected: PASS with no race report and no formatting output.

- [ ] **Step 8: Commit and exact-head verify**

Commit: `feat(pool): refresh bounded Stratum connections safely`

Require exact-head GPU-PoW v1 CI and generic CI PASS before Task 5.

---

### Task 5: No-listener guard, permanent CI gate and Stage E checkpoint

**Files:**
- Create: `pool/stratum/transport/source_guard_test.go`
- Modify: `.github/workflows/gpu-pow-v1-ci.yml`
- Modify: `docs/stratum/SUDHARMA_STRATUM_V1.md`
- Modify: `docs/superpowers/plans/2026-08-29-stratum-connection-transport.md`

**Interfaces:**
- Consumes: all Stage E transport APIs plus existing Stage D transcript/RPC tests.
- Produces: repository-enforced no-listener evidence, `Stage E bounded Stratum transport gate`, operator-facing Stage E boundary documentation and final checkpoint metadata.

- [ ] **Step 1: Write failing architectural source guard**

Parse non-test `.go` files in `pool/stratum/transport` with `go/parser`. Walk `ast.CallExpr` nodes and reject selector calls whose package/name resolve textually to any of:

```text
net.Listen
tls.Listen
http.Serve
http.ListenAndServe
http.ListenAndServeTLS
```

Also reject any production declaration whose type is `net.Listener`. The test must pass only because Stage E consumes `net.Conn` and has no accept-loop API.

To prove the test can fail, place the forbidden-call matcher in a helper and unit-test it against a parsed in-memory source snippet containing `net.Listen("tcp", ":1234")` before scanning the real directory.

- [ ] **Step 2: Run source-guard tests and verify RED**

Run: `go test ./pool/stratum/transport -run 'TestForbiddenListenerMatcher' -count=1`

Expected: initial failing matcher assertion until the AST guard helper is implemented; after the helper is implemented, run the real-tree guard and keep it green.

- [ ] **Step 3: Implement/verify the no-listener guard**

Complete the AST matcher and real-directory scan, excluding `_test.go` files from the production scan. Then run:

```bash
go test ./pool/stratum/transport -run 'TestForbiddenListenerMatcher|TestTransportHasNoListener' -count=1 -v
```

Expected: PASS and no listener primitive detected.

- [ ] **Step 4: Add the permanent Stage E workflow gate**

Immediately after the existing Stage D offline Stratum protocol gate, add exactly:

```yaml
      - name: Stage E bounded Stratum transport gate
        run: go test -race ./pool/stratum/... ./rpc -run 'Stratum|Transport|OfflineStratumTranscript' -count=1 -v
```

Do not remove or weaken the Stage D gate, activation/rollback rehearsal, disabled-default gate, full regression or build steps.

- [ ] **Step 5: Update operator protocol documentation**

Add a `Stage E injected connection transport` section to `docs/stratum/SUDHARMA_STRATUM_V1.md` recording:

- `ServeConn` accepts an already-open connection and owns/closes it;
- LF/CRLF, 64 KiB stream bound, finite read/write deadlines;
- immediate + periodic immutable work refresh after authorization;
- serialized writes, protocol-error budget and per-connection token bucket;
- no listener/TLS/public endpoint/IP policy/payout/vardiff/Kryptex-specific extension;
- passing Stage E remains software-only evidence, not physical GPU or Kryptex approval evidence.

- [ ] **Step 6: Run complete Stage E verification**

Run:

```bash
gofmt -w pool/stratum/transport/*.go pool/stratum/*.go rpc/stratum_adapter*.go
go vet ./...
go test ./... -count=1
go test -race ./pool/stratum/... ./rpc -count=1
go test ./pool/stratum -run TestOfflineStratumTranscript -count=1 -v
go test ./pow -run TestGPUV1NetworkActivationDefaultsRemainDisabled -count=1 -v
if grep -R -nE 'net\.Listen\(|tls\.Listen\(|http\.ListenAndServe(TLS)?\(|http\.Serve\(' pool/stratum/transport --include='*.go' --exclude='*_test.go'; then
  echo 'listener primitive found in Stage E production code'
  exit 1
fi
```

Expected: all commands PASS, source grep prints no production match, Stage D transcript remains byte-for-byte green, and activation defaults remain disabled.

- [ ] **Step 7: Mark this plan complete and commit**

Mark every completed checkbox `[x]` and add an execution checkpoint with final commit/run numbers without changing the safety boundary.

Commit: `test(pool): gate bounded Stratum connection transport`

- [ ] **Step 8: Verify exact live head and update PR #25**

Require final exact-head GPU-PoW v1 CI and generic CI PASS. Confirm PR #25 remains open/draft/unmerged, then update its checkpoint with Stage E commit/run numbers and these remaining independent gates: RTX 2060 localhost physical round trip; AMD/non-NVIDIA OpenCL ≥4 GiB evidence; public listener/TLS/admission review; vardiff if required by onboarding; payout/accounting/custody design if pursued; Kryptex profile validation and Kryptex-side approval; later explicit consensus-deployment decision.
