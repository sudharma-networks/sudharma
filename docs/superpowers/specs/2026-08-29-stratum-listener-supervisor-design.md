# Bounded Stratum Listener Supervisor Design

## Status

Stage F design for Sudharma mining-pool interoperability.

Stage F sits above the completed Stage E injected-connection transport. It adds a bounded accept/admission/TLS supervisor around an injected `net.Listener`, but it still does **not** call `net.Listen`, bind a port, load certificate files, expose a public endpoint, wire Stratum into `cmd/sudharma-rpcd`, deploy to Seed-1/Seed-2, or activate GPU-PoW.

The Stage D protocol core and Stage E per-connection transport remain the source of truth for worker identity, immutable jobs, nonce lanes, share classification, request framing, per-connection rate limits, protocol-error budgets, deadlines and work refresh.

## Goal

Add the smallest production-shaped listener boundary needed to exercise real accept-loop, aggregate admission and optional TLS behavior without authorizing a public service. The new layer must accept an already-created listener, apply finite global and per-IP connection caps, optionally perform a bounded TLS handshake, delegate each admitted stream to Stage E `transport.ServeConn`, and shut down deterministically with no leaked goroutines.

## Compatibility context

Current Kryptex Pool guidance continues to use Stratum TCP endpoints, commonly accepts `WALLET_ADDRESS.WORKER_NAME`, accepts password `x` or blank, and offers SSL/TLS where supported by the miner. Sudharma Stage D already covers the required wallet/worker and password shape. Stage F therefore changes no protocol messages or mining semantics.

Kryptex-side onboarding and any exact endpoint/profile requirements remain external approval gates. Stage F is only a software boundary that makes later interoperability testing possible.

## Selected architecture

Three approaches were considered.

1. **Injected listener supervisor — selected.** `ServeListener` receives an existing `net.Listener`; the caller remains responsible for binding addresses and loading certificates. This lets Stage F test a real accept loop and TLS while structurally preventing the package from opening a network port by itself.
2. **Socket-owning server — deferred.** Calling `net.Listen` or `tls.Listen` here would cross the deployment/exposure gate too early and mix address policy, certificate loading and service startup with transport correctness.
3. **Direct node integration — rejected.** Wiring Stratum into `cmd/sudharma-rpcd` before physical GPU gates, admission review and Kryptex profile validation would make an unfinished pool surface part of the node lifecycle.

## Package boundary

Create `pool/stratum/server`.

Dependencies are one-way:

```text
pool/stratum
    ^
    |
pool/stratum/transport
    ^
    |
pool/stratum/server
```

The Stage F package may import `crypto/tls`, `context`, `errors`, `fmt`, `net`, `sync`, `time`, and `pool/stratum/transport`. It must not import node, blockchain, RPC server, wallet, filesystem certificate loaders or deployment packages.

Public surface:

```go
type Config struct {
    MaxConnections      int
    MaxConnectionsPerIP int
    TLSConfig           *tls.Config
    TLSHandshakeTimeout time.Duration
    AcceptErrorBackoff  time.Duration
}

func ServeListener(
    ctx context.Context,
    listener net.Listener,
    factory transport.SessionFactory,
    transportConfig transport.Config,
    config Config,
) error
```

`ServeListener` owns the injected listener for the duration of the call and closes it before returning. Accepted connections are owned by Stage E after admission.

## Configuration

All listener-level controls are finite. Zero values select conservative Stage F defaults:

- maximum concurrent connections: 256;
- maximum concurrent connections per source IP: 8;
- TLS handshake timeout: 10 seconds;
- temporary accept-error backoff: 100 milliseconds.

Negative integer limits and negative durations are invalid. A nil listener or nil session factory is invalid.

`MaxConnections` and `MaxConnectionsPerIP` must resolve to positive values, and per-IP capacity cannot exceed the global capacity.

`TLSConfig == nil` means the injected listener carries plaintext Stratum. A non-nil TLS config means each admitted connection is wrapped with `tls.Server` inside Stage F before Stage E sees it.

Stage F clones the supplied TLS configuration before use and never mutates caller-owned configuration.

If `TLSConfig.MinVersion == 0`, Stage F sets the clone to `tls.VersionTLS12`. If the supplied minimum is below TLS 1.2, configuration is rejected. TLS 1.3 remains allowed. Stage F does not force cipher suites beyond Go's secure defaults and does not implement client-certificate authentication in this checkpoint.

Stage F never loads certificate/key files. The caller must supply a usable `tls.Config`; a handshake failure is connection-local and must not terminate the entire listener unless the parent context is canceled or the listener itself becomes unusable.

## Source identity and proxy policy

Admission uses only `conn.RemoteAddr()` from the accepted connection.

For TCP-style addresses, the canonical admission key is the parsed IP string. IPv4-mapped IPv6 addresses are normalized to the corresponding IPv4 form so one client cannot bypass a per-IP cap through textual address variants.

If an address is not a `*net.TCPAddr`, Stage F falls back to the full `RemoteAddr().String()` as a bounded in-memory admission key. The key is never emitted in returned errors.

Stage F explicitly does **not** trust `X-Forwarded-For`, PROXY protocol, reverse-proxy headers, or miner-supplied identity for admission. Trusted-proxy support is a later operational stage because it requires an explicit trust boundary.

## Admission model

Maintain one listener-local admission tracker protected by a mutex:

```go
type admission struct {
    maxTotal int
    maxPerIP int
    total    int
    byIP     map[string]int
}
```

For every accepted connection:

1. derive the source key;
2. atomically attempt to reserve one global slot and one source slot;
3. if either cap is full, close the connection immediately without creating a Stage D session and continue accepting;
4. if admitted, spawn exactly one connection goroutine;
5. release both slots exactly once when that goroutine exits.

Rejected connections receive no Stratum response. Admission occurs before any TLS handshake and before any session factory call so resource-expensive work cannot bypass the caps.

A cap rejection is normal listener operation, not a fatal server error.

## Accept loop and temporary errors

`ServeListener` calls only `listener.Accept()`; it must never create another listener.

A successful accept is immediately passed through admission.

If `Accept` returns because the parent context was canceled and the cancellation watcher closed the listener, return `ctx.Err()`.

Temporary accept errors are retried after `AcceptErrorBackoff`. The backoff is fixed for this checkpoint to keep behavior deterministic and bounded; exponential retry policy is unnecessary here.

A non-temporary accept error terminates `ServeListener` with operation-only context such as `accept Stratum connection`.

The cancellation watcher closes the injected listener when the parent context ends so a blocked `Accept()` wakes promptly.

## TLS handshake lifecycle

For admitted connections with TLS enabled:

1. clone/wrap the raw connection using `tls.Server`;
2. set a full connection deadline of `now + TLSHandshakeTimeout`;
3. call `HandshakeContext(serveCtx)`;
4. on success, clear the temporary handshake deadline with `SetDeadline(time.Time{})`;
5. delegate the `*tls.Conn` to `transport.ServeConn`.

Handshake failure closes the connection and releases admission slots without invoking the Stage D session factory. It does not terminate the listener.

The TLS wrapper and Stage E share ownership safely because Stage E owns and closes the connection once delegation begins; Stage F must not double-close as part of normal delegation cleanup.

## Connection execution and error containment

Each admitted connection runs independently.

The connection goroutine may return because of Stage E rate limiting, protocol budget exhaustion, malformed input, I/O timeout, refresh failure, TLS failure or normal EOF. These are connection-local outcomes and do not bring down the listener.

Stage F does not log raw worker identity, wallet address, request body, password, nonce, remote IP or TLS peer material. This package exposes no logging API in Stage F.

Only listener-level failures are returned by `ServeListener`: invalid configuration, unrecoverable accept failure, or parent context cancellation.

A future observability stage may add sanitized counters, but no callback/hook interface is added now.

## Shutdown

`ServeListener` creates a derived context and one cancellation watcher that closes the listener when canceled.

Connection goroutines inherit the derived context. When the accept loop terminates:

1. cancel the derived context;
2. close the listener if not already closed;
3. wait for every admitted connection goroutine to return;
4. wait for the cancellation watcher to return;
5. then return the listener-level result.

Because Stage E responds to context cancellation by waking blocked connection I/O, listener shutdown must not leave a connection goroutine behind.

The package must be race-clean under concurrent accepts, cap rejections, TLS handshakes and shutdown.

## Error model

Expose stable Stage F policy errors only where callers need configuration decisions:

```go
var ErrInvalidConfig = errors.New("invalid Stratum server configuration")
```

Connection-local errors are intentionally contained and are not surfaced as the listener return value.

Unrecoverable listener errors wrap the underlying value with operation-only context. Returned errors must not contain remote addresses or miner-controlled request content.

## Security invariants

Stage F must prove:

- no production call to `net.Listen`, `tls.Listen`, `http.Serve`, `http.ListenAndServe`, `http.ListenAndServeTLS` or equivalent listener creation exists in `pool/stratum/server`;
- `ServeListener` accepts only an injected listener;
- admission occurs before TLS and before session creation;
- global and per-source caps cannot be exceeded under concurrent accepts;
- textual IPv4/IPv6 variants cannot trivially bypass the source cap;
- cap rejection closes only the rejected stream and leaves the listener alive;
- TLS below 1.2 is rejected at configuration time;
- zero TLS minimum is normalized to TLS 1.2 on a cloned config;
- TLS handshake timeout is finite;
- a failed TLS handshake does not create a Stratum session;
- a failed or abusive Stage E connection does not terminate the listener;
- cancellation wakes a blocked accept and all active Stage E connections;
- all admission slots are released on every connection exit path;
- no Stage F goroutine outlives `ServeListener`;
- Stage D/E behavior and their permanent CI gates remain unchanged;
- no network GPU-PoW activation default changes.

## Test strategy

Prefer deterministic fake listeners/connections for admission and accept-error tests, `net.Pipe` for connection lifecycle tests, and an in-memory self-signed certificate generated only in tests for TLS handshake behavior.

Required tests:

- configuration defaults and invalid limits/durations;
- TLS config cloning and TLS 1.2 minimum enforcement;
- global cap rejects the next connection before factory invocation;
- per-IP cap rejects a same-IP connection while allowing a different IP;
- admission release permits a later connection after an earlier one exits;
- concurrent admission never exceeds configured totals under `-race`;
- plaintext admitted connection reaches Stage E subscribe/authorize flow;
- valid TLS client reaches the same Stage E flow;
- TLS handshake timeout/failure remains connection-local and does not invoke the session factory;
- Stage E `ErrRateLimited`/protocol failure on one connection does not stop another connection;
- temporary accept error is retried;
- permanent accept error terminates with `accept Stratum connection` context;
- context cancellation closes the listener, cancels active sessions and joins all goroutines;
- production source guard finds no listener-creation primitive;
- existing Stage D offline transcript and Stage E transport tests remain unchanged and green.

## CI gate

Add a Stage F gate after the existing Stage E gate:

```yaml
- name: Stage F bounded Stratum listener supervisor gate
  run: go test -race ./pool/stratum/... ./rpc -run 'Stratum|Transport|Server|OfflineStratumTranscript' -count=1 -v
```

During Stage F development, the dedicated branch may be included in the GPU-PoW workflow push filter so RED/GREEN commits can be proven by GitHub Actions. That branch-only workflow allowance must not weaken any existing CI step.

## Explicit exclusions

Stage F does not implement or authorize:

- `net.Listen` or direct port binding;
- certificate/key file loading;
- DNS/domain configuration;
- public firewall/security-group changes;
- reverse proxy or PROXY-protocol trust;
- IP reputation/geo blocking or DDoS mitigation claims;
- variable share difficulty/vardiff;
- payout accounting, balances, fees, thresholds or custody;
- persistent miner accounts;
- Kryptex-specific protocol extensions or a listing/approval claim;
- miner package configuration changes;
- `cmd/sudharma-rpcd` Stratum wiring;
- Seed-1/Seed-2 deployment;
- any finite GPU-PoW activation height;
- unrestricted GPU mining;
- PR #25 merge;
- mainnet activation.

Those remain later independent gates. Stage F is software-only listener supervision evidence and is not physical GPU evidence, a public pool launch, or Kryptex approval.
