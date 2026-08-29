# Bounded Stratum Listener Supervisor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a bounded injected-listener supervisor above Stage E that enforces aggregate admission, optional TLS and deterministic shutdown without binding a port or exposing Stratum publicly.

**Architecture:** New package `pool/stratum/server` receives an already-created `net.Listener`, performs listener-level admission and optional TLS handshakes, and delegates every admitted stream to existing `transport.ServeConn`. The server package never calls `net.Listen`, never loads certificate files and never wires itself into the node.

**Tech Stack:** Go 1.26.6 standard library, existing `pool/stratum`, `pool/stratum/transport`, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-08-29-stratum-listener-supervisor-design.md`

## Global Constraints

- Do not call `net.Listen`, `tls.Listen`, `http.Serve`, `http.ListenAndServe` or `http.ListenAndServeTLS` from Stage F production code.
- Do not wire Stratum into `cmd/sudharma-rpcd`.
- Do not load TLS certificate/key files in Stage F; accept only caller-supplied `*tls.Config`.
- Do not add PROXY protocol or trust reverse-proxy headers.
- Do not add vardiff, accounting, payouts, fees, custody or persistent miner accounts.
- Do not deploy to Seed-1/Seed-2, set a finite GPU-PoW activation height, enable unrestricted GPU mining, merge PR #25 or activate mainnet.
- Preserve Stage D worker/job/share semantics and Stage E per-connection framing, limiter, deadline and refresh behavior unchanged.
- Every production behavior begins with a failing test and is committed only after focused and repository CI evidence is green.

---

### Task 1: Stage F configuration and TLS policy

**Files:**
- Create: `pool/stratum/server/config.go`
- Test: `pool/stratum/server/config_test.go`
- Modify: `.github/workflows/gpu-pow-v1-ci.yml`

**Interfaces:**
- Produces: `Config`, `normalizedConfig`, `normalizeConfig(Config) (normalizedConfig, error)`, `ErrInvalidConfig`.
- Consumes: `crypto/tls`, `time.Duration`.

- [ ] **Step 1: Enable CI on the isolated Stage F branch**

Add `feature/gpu-pow-v1-stage-f` to the existing GPU-PoW workflow push branch filter. Do not alter any test/build step. This is branch-development plumbing only and precedes production behavior.

- [ ] **Step 2: Write failing configuration tests**

Cover exact zero-value defaults:

```go
const (
    defaultMaxConnections      = 256
    defaultMaxConnectionsPerIP = 8
    defaultTLSHandshakeTimeout = 10 * time.Second
    defaultAcceptErrorBackoff  = 100 * time.Millisecond
)
```

Prove:

- zero values resolve to all four defaults;
- negative connection limits are rejected;
- negative TLS/backoff durations are rejected;
- per-IP capacity above global capacity is rejected;
- `TLSConfig == nil` remains nil;
- a caller TLS config with `MinVersion == 0` is cloned and normalized to `tls.VersionTLS12` without mutating the caller;
- TLS minimum below 1.2 is rejected;
- TLS 1.2 and TLS 1.3 are accepted.

Representative test:

```go
func TestNormalizeConfigClonesTLSAndEnforcesMinimum(t *testing.T) {
    original := &tls.Config{}
    got, err := normalizeConfig(Config{TLSConfig: original})
    if err != nil { t.Fatal(err) }
    if got.tlsConfig == original { t.Fatal("TLS config was not cloned") }
    if got.tlsConfig.MinVersion != tls.VersionTLS12 { t.Fatalf("min=%x", got.tlsConfig.MinVersion) }
    if original.MinVersion != 0 { t.Fatalf("caller TLS config mutated: %x", original.MinVersion) }
}
```

- [ ] **Step 3: Verify RED with GitHub Actions**

Commit only the tests plus branch-filter workflow change with message:

`test(pool): define Stage F server configuration contract`

Expected exact-head GPU-PoW CI: FAIL because `pool/stratum/server` production configuration symbols do not exist. Confirm the failure is compilation from the intended missing API, not workflow syntax or unrelated regression.

- [ ] **Step 4: Implement minimal configuration normalization**

Public shape:

```go
type Config struct {
    MaxConnections      int
    MaxConnectionsPerIP int
    TLSConfig           *tls.Config
    TLSHandshakeTimeout time.Duration
    AcceptErrorBackoff  time.Duration
}
```

Internal shape:

```go
type normalizedConfig struct {
    maxConnections      int
    maxConnectionsPerIP int
    tlsConfig           *tls.Config
    tlsHandshakeTimeout time.Duration
    acceptErrorBackoff  time.Duration
}
```

Rules are exactly those in Step 2. Define:

```go
var ErrInvalidConfig = errors.New("invalid Stratum server configuration")
```

- [ ] **Step 5: Verify GREEN**

Require exact-head GPU-PoW CI success, then confirm focused job output includes:

```bash
go test -race ./pool/stratum/... ./rpc -run 'Stratum|Transport|Server|OfflineStratumTranscript' -count=1 -v
```

Commit message:

`feat(pool): define bounded Stratum server configuration`

---

### Task 2: Source-key normalization and atomic admission

**Files:**
- Create: `pool/stratum/server/admission.go`
- Test: `pool/stratum/server/admission_test.go`

**Interfaces:**
- Consumes: normalized positive `maxConnections`, `maxConnectionsPerIP`, `net.Addr`.
- Produces: `sourceKey(net.Addr) string`, internal `admission`, `newAdmission(total, perIP int) *admission`, `(*admission).Acquire(key string) bool`, `(*admission).Release(key string)`.

- [ ] **Step 1: Write failing source-key tests**

Prove:

- `127.0.0.1:1000` and `127.0.0.1:2000` produce the same key;
- IPv4-mapped IPv6 normalizes to the IPv4 string;
- native IPv6 ignores port but preserves the canonical IP;
- non-TCP addresses fall back to `Addr.String()`.

Use a tiny test-only `net.Addr` implementation for the fallback case.

- [ ] **Step 2: Write failing admission tests**

Prove sequentially:

```go
a := newAdmission(2, 1)
if !a.Acquire("ip-a") { t.Fatal("first admission rejected") }
if a.Acquire("ip-a") { t.Fatal("per-IP limit bypassed") }
if !a.Acquire("ip-b") { t.Fatal("different IP rejected") }
if a.Acquire("ip-c") { t.Fatal("global limit bypassed") }
a.Release("ip-a")
if !a.Acquire("ip-c") { t.Fatal("released slot not reusable") }
```

Add a concurrent test with many goroutines that records the maximum successful in-flight reservations and proves it never exceeds either configured limit under `-race`.

Release of an absent key must not underflow counts; tests should call it and then verify future valid acquisitions still behave correctly.

- [ ] **Step 3: Verify RED**

Commit tests only:

`test(pool): define Stage F connection admission contract`

Expected CI: FAIL because admission/source-key production APIs are missing.

- [ ] **Step 4: Implement minimal admission tracker**

Use one mutex and `map[string]int`. `Acquire` checks total and per-key count atomically before incrementing. `Release` decrements only existing positive reservations, removes zero map entries and never lets total become negative.

For `*net.TCPAddr`, normalize IP using `To4()` first, else `IP.String()`. Ignore source port.

- [ ] **Step 5: Verify GREEN**

Require exact-head GPU-PoW CI PASS and no race report.

Commit:

`feat(pool): bound Stratum listener admission`

---

### Task 3: Injected plaintext accept loop and connection containment

**Files:**
- Create: `pool/stratum/server/server.go`
- Test: `pool/stratum/server/server_test.go`
- Test helper: `pool/stratum/server/test_helpers_test.go`

**Interfaces:**
- Consumes: `net.Listener`, `transport.SessionFactory`, `transport.Config`, Stage F `Config`, admission from Task 2.
- Produces:

```go
func ServeListener(
    ctx context.Context,
    listener net.Listener,
    factory transport.SessionFactory,
    transportConfig transport.Config,
    config Config,
) error
```

- [ ] **Step 1: Build deterministic fake-listener test helpers**

Test-only listener supports queued `Accept` results and records `Close`. Test-only connections expose configurable `RemoteAddr` values. Do not place test fakes in production files.

- [ ] **Step 2: Write failing validation and cap tests**

Prove nil listener and nil factory return `ErrInvalidConfig`.

With global max 1, queue two connections while the first remains open. Assert second is closed before its Stage D session factory is invoked. Repeat for per-IP max 1; then prove a different IP is admitted.

- [ ] **Step 3: Write failing plaintext delegation test**

Use a real `net.Pipe`-backed listener fixture that yields one server side with a TCP-style remote address. Construct the same fixed Stage D session fixture pattern used by Stage E tests. Send `mining.subscribe` and `mining.authorize`; prove Stage E response framing and immediate work notification are received unchanged.

The Stage F test must assert only supervision/delegation behavior and not duplicate Stage D protocol assertions unnecessarily.

- [ ] **Step 4: Verify RED**

Commit tests only:

`test(pool): define injected Stratum listener lifecycle`

Expected CI: FAIL because `ServeListener` is missing.

- [ ] **Step 5: Implement minimal plaintext ServeListener**

Behavior:

1. validate arguments and normalize config;
2. create `serveCtx` and cancellation watcher;
3. watcher closes the injected listener when context ends;
4. accept loop derives source key and tries admission before factory/session work;
5. rejected streams are closed immediately and loop continues;
6. admitted streams start one goroutine each;
7. goroutine calls `transport.ServeConn(serveCtx, conn, factory, transportConfig)` and contains every returned connection-local error;
8. goroutine releases admission exactly once;
9. accept loop returns only cancellation or unrecoverable listener failure;
10. defer cancels, closes listener, waits for connection goroutines and watcher.

No production logging/hook API.

- [ ] **Step 6: Verify GREEN and Stage E preservation**

Require exact-head GPU-PoW CI PASS, including existing Stage D and Stage E gates.

Commit:

`feat(pool): supervise injected Stratum listeners`

---

### Task 4: TLS handshake boundary and accept-error policy

**Files:**
- Create: `pool/stratum/server/tls.go`
- Modify: `pool/stratum/server/server.go`
- Test: `pool/stratum/server/tls_test.go`
- Test: `pool/stratum/server/accept_test.go`

**Interfaces:**
- Consumes: normalized optional `*tls.Config`, `tlsHandshakeTimeout`, `acceptErrorBackoff`.
- Produces internal `prepareConn(context.Context, net.Conn, normalizedConfig) (net.Conn, error)` and temporary-accept retry behavior.

- [ ] **Step 1: Write failing TLS success test**

Generate an in-memory self-signed test certificate with standard library `crypto/x509`; do not commit any private key or certificate fixture.

Use `net.Pipe`, `tls.Client`, and Stage F server TLS configuration. Prove a TLS 1.2+ client can complete subscribe/authorize and receive the same Stage E messages.

- [ ] **Step 2: Write failing TLS failure/timeout tests**

Prove:

- a plaintext client connected to a TLS-enabled Stage F connection fails the handshake and never invokes the Stage D session factory;
- a client that sends nothing hits the finite handshake deadline;
- after either failure, a later queued valid connection is still accepted, proving handshake errors are connection-local.

Use deadlock-guard selects only; protocol timing assertion comes from the configured handshake timeout and server completion behavior.

- [ ] **Step 3: Write failing accept-error tests**

Create a temporary `net.Error` fixture. Queue temporary error -> valid connection -> context cancellation and prove the valid connection is eventually accepted. Queue a permanent error and prove `ServeListener` returns an error wrapping it with text `accept Stratum connection`.

- [ ] **Step 4: Verify RED**

Commit tests only:

`test(pool): define Stage F TLS and accept policy`

Expected CI: FAIL because TLS preparation and temporary accept retry are missing.

- [ ] **Step 5: Implement bounded TLS preparation**

When TLS is disabled return the raw connection unchanged.

When enabled:

```go
secure := tls.Server(conn, normalized.tlsConfig)
if err := secure.SetDeadline(time.Now().Add(normalized.tlsHandshakeTimeout)); err != nil { ... }
if err := secure.HandshakeContext(ctx); err != nil { ... }
if err := secure.SetDeadline(time.Time{}); err != nil { ... }
return secure, nil
```

Wrap errors with operation-only contexts such as `set Stratum TLS handshake deadline`, `handshake Stratum TLS`, and `clear Stratum TLS handshake deadline`. The caller closes failed raw/TLS connections and releases admission without session creation.

- [ ] **Step 6: Implement temporary accept retry**

If the accept error implements `net.Error` and `Temporary()` is true, wait for either `time.After(acceptErrorBackoff)` or context cancellation, then retry. Do not busy-loop.

Permanent errors terminate the listener. Context cancellation always wins when it caused listener closure.

- [ ] **Step 7: Verify GREEN**

Require exact-head GPU-PoW CI PASS and race-clean Stage F tests.

Commit:

`feat(pool): bound Stratum TLS and accept failures`

---

### Task 5: Shutdown proof, no-listener guard, permanent CI gate and Stage F checkpoint

**Files:**
- Test: `pool/stratum/server/shutdown_test.go`
- Create: `pool/stratum/server/source_guard_test.go`
- Modify: `.github/workflows/gpu-pow-v1-ci.yml`
- Modify: `docs/stratum/SUDHARMA_STRATUM_V1.md`
- Modify: `docs/superpowers/plans/2026-08-29-stratum-listener-supervisor.md`

**Interfaces:**
- Consumes: complete Stage F server plus existing Stage D/E gates.
- Produces: cancellation/goroutine-join proof, architectural source guard, permanent Stage F CI gate, operator boundary documentation and final checkpoint metadata.

- [ ] **Step 1: Write failing shutdown tests**

Prove cancellation while blocked in `Accept` closes the listener and returns `context.Canceled`.

Prove cancellation with multiple admitted Stage E connections causes all active connection goroutines to return before `ServeListener` returns. Use a test listener/connection wrapper that increments/decrements an atomic active counter and assert it is zero at return.

Prove admission slots are released after normal EOF, Stage E rate-limit/protocol termination and TLS failure so a later connection can enter.

- [ ] **Step 2: Verify RED if any shutdown guarantee is missing**

Run via exact-head CI. If tests already pass because Task 3/4 implementation satisfies the guarantees, strengthen only with an invariant not previously exercised; do not manufacture an artificial failure. Record the observed behavior in the plan checkpoint.

- [ ] **Step 3: Add architectural no-bind guard**

Parse non-test `.go` files in `pool/stratum/server` and fail on production selector calls to:

```text
net.Listen
tls.Listen
http.Serve
http.ListenAndServe
http.ListenAndServeTLS
```

Also reject production declarations that expose a socket-owning helper whose name begins with `ListenAndServe`.

Unit-test the matcher against an in-memory forbidden source snippet, then scan the real directory.

- [ ] **Step 4: Add permanent Stage F workflow gate**

Immediately after Stage E add:

```yaml
      - name: Stage F bounded Stratum listener supervisor gate
        run: go test -race ./pool/stratum/... ./rpc -run 'Stratum|Transport|Server|OfflineStratumTranscript' -count=1 -v
```

Keep the Stage F development branch in the push filter until final integration is proven. Do not remove or weaken Stage D/E gates, activation/rollback rehearsal, disabled-default guard, full regression or build.

- [ ] **Step 5: Update operator documentation**

Add `Stage F injected listener supervisor` to `docs/stratum/SUDHARMA_STRATUM_V1.md` documenting:

- accepts an already-created listener only;
- global 256 and per-source 8 default caps;
- no proxy trust;
- optional caller-supplied TLS config with TLS 1.2 minimum and 10s handshake timeout;
- connection-local failure containment;
- clean cancellation and goroutine joining;
- still no bind address, certificate loading, node wiring, public endpoint, vardiff, payout/accounting or Kryptex approval claim.

- [ ] **Step 6: Complete exact-head verification**

Require on the same final SHA:

- GPU-PoW v1 CI: PASS;
- generic CI if triggered for the branch/PR: PASS;
- Stage D offline gate: PASS;
- Stage E bounded transport gate: PASS;
- Stage F listener supervisor gate: PASS;
- disabled network activation defaults: PASS;
- full repository regression: PASS;
- node build: PASS.

- [ ] **Step 7: Mark the Stage F plan complete**

Check every proven item and add execution checkpoint SHAs/run numbers. Commit:

`test(pool): gate bounded Stratum listener supervisor`

- [ ] **Step 8: Integrate only after exact-head green**

Fast-forward `feature/gpu-pow-v1` to the final Stage F SHA only after all required checks are green. Do not merge PR #25. Let the existing PR update naturally from its feature branch, then require exact-head PR CI green again.

Update PR #25 and issue #13 with the Stage F checkpoint and preserve remaining gates: physical RTX 2060 localhost round trip, AMD/non-NVIDIA OpenCL 4 GiB+ evidence, vardiff decision, pool accounting/custody design if pursued, final public bind/deployment review, Kryptex profile validation/approval and later consensus activation decision.
