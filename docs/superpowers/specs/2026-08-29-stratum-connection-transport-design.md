# Bounded Stratum Connection Transport Design

## Status

Stage E design for the next offline-only Sudharma mining-pool checkpoint.

This stage sits above the completed Stage D `pool/stratum` protocol core and below any future public listener. It accepts an already-open `net.Conn`; it does **not** call `net.Listen`, bind a port, terminate TLS, inspect proxy headers, expose a public endpoint, wire into `cmd/sudharma-rpcd`, deploy to Seed-1/Seed-2, or activate GPU-PoW.

## Goal

Provide a production-shaped but non-listening connection transport that can safely carry the frozen Sudharma Stratum profile over one injected stream. The transport must enforce bounded newline framing, deadlines, serialized writes, protocol-error framing, authorization-triggered work delivery, periodic immutable work refresh, cancellation and per-connection abuse limits without changing Stage D consensus or share semantics.

## Why this approach

Three approaches were considered:

1. **Injected `net.Conn` transport core — selected.** It exercises real stream behavior and is compatible with plain TCP or a future `*tls.Conn`, while structurally preventing this stage from opening a network listener.
2. **Full `net.Listener` server — deferred.** It would prematurely cross the public-exposure gate and require TLS, accept-loop limits, IP/proxy policy and deployment review in the same change.
3. **HTTP/WebSocket transport — rejected for this stage.** It does not match the intended Stratum-style pool path and would add a second protocol surface without solving Kryptex-oriented TCP interoperability.

## Package boundary

Create subpackage `pool/stratum/transport`.

The subpackage imports `pool/stratum`; the core package must never import the transport package. `rpc` remains outside both layers. The transport receives a factory that constructs one fully configured Stage D `*stratum.Session` per connection.

Proposed public surface:

```go
type SessionFactory func() (*stratum.Session, error)

type Config struct {
    ReadTimeout       time.Duration
    WriteTimeout      time.Duration
    RefreshInterval   time.Duration
    MaxProtocolErrors int
    RequestsPerSecond uint32
    Burst             uint32
    Clock             Clock // optional test clock; real clock by default
}

func ServeConn(ctx context.Context, conn net.Conn, factory SessionFactory, cfg Config) error
```

`ServeConn` owns the passed connection for the duration of the call and closes it before returning. No accept loop is part of this package.

## Configuration

All resource controls must be finite. Zero values use conservative defaults suitable for an interoperability checkpoint, not a production deployment recommendation:

- read timeout: 30 seconds;
- write timeout: 10 seconds;
- authorized work refresh interval: 5 seconds;
- maximum protocol errors before close: 8;
- request rate: 20 requests/second;
- burst: 40 requests.

Negative durations, zero/invalid explicit limits, or a nil session factory/connection fail construction/entry immediately. Tests use injected clock/ticker behavior where timing determinism matters.

These defaults may be changed only in a later deployment review; no public service is authorized by choosing them here.

## Framing

Client messages are newline-delimited UTF-8 JSON requests matching Stage D.

The transport reads at most 64 KiB of request content plus the line terminator. It must never use an unbounded `ReadString`/`ReadAll` path. Both `\n` and `\r\n` line endings are accepted; the terminator is removed before calling `Session.Handle`.

A line whose content exceeds 64 KiB is rejected and the connection is closed after a best-effort `-32600 invalid request` response with `id:null`. An EOF with a non-empty unterminated final line is processed only if that line is within the size limit; an empty EOF ends the connection cleanly.

Malformed JSON within a bounded line is converted from the Stage D `*ProtocolError` into a normal response with `id:null`. Recoverable protocol errors do not automatically close the stream, but each one consumes the per-connection protocol-error budget.

## Request processing

For each complete request line:

1. enforce the per-connection token bucket before protocol processing;
2. decode enough to identify the method using the existing Stage D decoder;
3. call `Session.Handle` exactly once for the request;
4. frame any returned `ProtocolError` as a response with `id:null`;
5. serialize all returned messages through one connection writer;
6. after a successful `mining.authorize`, immediately call `Session.RefreshWork` and write the resulting `mining.set_difficulty` and `mining.notify` messages.

The transport must not reinterpret worker identity, nonce lanes, share difficulty, job IDs or submit results. Those remain exclusively Stage D responsibilities.

## Work refresh pump

After successful authorization, one refresh loop periodically calls `Session.RefreshWork` using the same session. Identical source work emits no message because Stage D already suppresses it. Changed immutable work emits the frozen difficulty/notify pair.

Only one refresh loop may run per connection. Repeated authorization of the same identity must not create additional loops.

All writes from request handling and refresh delivery pass through a single mutex-protected writer so JSON lines never interleave.

A refresh source error, write error or context cancellation terminates the connection. This is intentionally fail-closed for the offline checkpoint; retry/backoff policy belongs to a later listener/operator stage.

## Deadlines and cancellation

Before each blocking read, set `ReadDeadline(now + ReadTimeout)`. Before each write batch, set `WriteDeadline(now + WriteTimeout)`. A timeout is returned as a transport error and closes the connection.

`ServeConn` must unblock promptly when the supplied context is canceled. A small cancellation goroutine may set an immediate deadline or close the owned connection; it must stop before `ServeConn` returns and must not leak.

No goroutine spawned by `ServeConn` may outlive the call.

## Per-connection rate limiting

Use a small token bucket local to one connection. It is not an IP-level DDoS defense and must not be documented as one.

- capacity = `Burst`;
- refill rate = `RequestsPerSecond` tokens/second;
- one complete request consumes one token before decoding;
- if no token is available, return a transport-level rate-limit error and close the connection without invoking `Session.Handle` for that request.

A future listener stage must add aggregate/IP/proxy-aware admission controls outside this package.

## Error model

Define sentinel/typed transport errors sufficient for tests and operator logs, including:

- invalid configuration;
- oversized line;
- protocol-error budget exceeded;
- rate limit exceeded;
- read timeout/error;
- write timeout/error;
- session-factory failure;
- work-refresh failure.

Do not expose secrets or raw connection metadata in error strings. Protocol parse/validation errors sent to a miner use the Stage D stable JSON-RPC codes.

## Security invariants

Stage E must prove all of the following:

- no call to `net.Listen`, `tls.Listen`, `http.Serve` or equivalent listener creation exists in `pool/stratum/transport`;
- a miner cannot bypass the 64 KiB Stage D request limit through stream fragmentation;
- fragmented and coalesced TCP-style writes preserve exact request boundaries;
- response/notification writes cannot interleave;
- only one session is created per connection;
- only one refresh pump starts after authorization;
- malformed requests are bounded by the protocol-error budget;
- rate-limited requests never reach the session;
- context cancellation and I/O errors leave no background goroutine running;
- Stage D immutable work and submission semantics are unchanged;
- network GPU-PoW activation defaults remain disabled.

## Test strategy

Use `net.Pipe` for end-to-end connection tests and small fake `net.Conn` implementations only where deadline/error injection is required.

Required tests include:

- subscribe/authorize over fragmented writes followed by immediate difficulty/notify;
- two requests coalesced in one client write;
- CRLF handling;
- exact 64 KiB boundary and 64 KiB+1 rejection;
- malformed JSON response and protocol-error budget exhaustion;
- request-rate exhaustion using a deterministic clock;
- duplicate authorization starts one refresh pump;
- changed work generates one new notification pair; identical work generates none;
- concurrent refresh/request outputs remain valid non-interleaved JSON lines;
- read deadline, write deadline and cancellation paths;
- no-listener source guard;
- existing `TestOfflineStratumTranscript` remains byte-for-byte unchanged;
- `go test -race ./pool/stratum/... ./rpc` passes.

## CI gate

Extend GPU-PoW CI with an offline transport gate after the Stage D protocol gate:

```yaml
- name: Stage E bounded Stratum transport gate
  run: go test -race ./pool/stratum/... ./rpc -run 'Stratum|Transport|OfflineStratumTranscript' -count=1 -v
```

The normal full regression, disabled-default activation test and generic race CI remain mandatory.

## Explicit exclusions

Stage E does not implement or authorize:

- `net.Listen`/public port binding;
- TLS certificate loading or termination;
- reverse proxy or PROXY-protocol trust;
- IP allow/deny rules or aggregate DDoS controls;
- vardiff;
- payout accounting, balances, fees or custody;
- persistent worker accounts;
- Kryptex-specific extensions or a claim of exact Kryptex wire compatibility;
- miner packaging changes;
- Seed-1/Seed-2 deployment;
- any finite testnet/mainnet GPU-PoW activation height;
- unrestricted GPU mining or mainnet activation.

Those are later, separately reviewed gates. Kryptex-side approval/configuration remains external to the Sudharma repository.
