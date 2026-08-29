# Sudharma Stratum V1 Protocol Profile

## Status

This document describes the staged Sudharma GPU-PoW Stratum interoperability profile. Stage D defines the transport-independent protocol core in `pool/stratum` and the immutable adapter in `rpc/stratum_adapter.go`. Stage E adds bounded framing and lifecycle for one already-open `net.Conn`. Stage F adds bounded supervision for an already-created `net.Listener`. Stage G adds a deliberately local-only real socket owner and end-to-end TCP/TLS compatibility evidence while preserving the Stage D/E/F contracts.

These checkpoints do **not** bind a public Stratum port, wire Stratum into `cmd/sudharma-rpcd`, deploy to Seed-1 or Seed-2, activate GPU-PoW, implement pool payouts or custody, or claim Kryptex approval/listing/wire compatibility.

## Wire model

One UTF-8 JSON object is processed per newline-delimited message. Client request IDs may be JSON strings or base-10 integers and are echoed exactly in responses. Batch requests, duplicate object keys, unknown top-level fields, unsupported methods, malformed UTF-8, trailing JSON values and messages above 64 KiB are rejected.

The Stage D core returns typed protocol errors. Stage E defines per-connection framing and bounded error handling; Stage F contains connection-local failures at the injected-listener boundary without changing protocol semantics.

## Client methods

| Method | Parameters | Result |
| --- | --- | --- |
| `mining.subscribe` | `[]` or `[agent]` | 32-character lowercase hexadecimal session ID |
| `mining.authorize` | `[WALLET.WORKER, password]` | `true` after successful binding |
| `mining.submit` | `[WALLET.WORKER, job_id, nonce_hex]` | one submit status string |

For `mining.authorize`, the password is not treated as a credential in this checkpoint. Only an empty string or literal `x` compatibility placeholder is accepted and discarded. Stage G exercises both forms over real loopback TCP sockets.

## Server notifications

`mining.set_difficulty` carries:

```text
[share_difficulty]
```

`mining.notify` carries these positional fields exactly:

```text
[job_id, algorithm, height, target_hex, header_prefix_hex,
 reward_address, version, network_difficulty, lane, clean_jobs]
```

A new source work ID creates a new generation and emits `clean_jobs=true`. Reusing the same work ID with changed immutable fields fails closed instead of emitting mutated work.

## Worker identity

The authorization username is exactly `WALLET.WORKER`.

- `WALLET` is exactly 40 lowercase hexadecimal characters.
- There is exactly one dot separator.
- `WORKER` is 1 to 32 ASCII characters from `[A-Za-z0-9_-]`.
- The full identity is at most 128 bytes.
- Whitespace, control characters, uppercase wallet hex, additional dots and other worker characters are rejected.
- One identity is bound to a session. Reauthorizing that session as a different identity fails closed.

## Nonce lanes

Sudharma GPU-PoW exposes one unsigned 64-bit nonce. A pool session receives a 32-bit lane for each current job:

```text
nonce = (lane << 32) | worker_counter
```

The submitted nonce is hexadecimal, contains 1 to 16 hex digits, has no `0x` prefix, and is decoded as an unsigned 64-bit value. The high 32 bits must equal the assigned lane; the miner searches the low 32 bits. A nonce outside the assigned lane returns `invalid`.

A lane is unique per work/session while current and is released when that work becomes stale.

## Share and block classification

Session share difficulty is separate from consensus network difficulty. The share target uses the same integer rule as Sudharma proof-of-work targeting:

```text
floor((2^256 - 1) / difficulty)
```

The immutable network target comes from the issued work template. The exact frozen work plus nonce is checked against the share target first and the network target second.

Submit result meanings are:

| Result | Meaning |
| --- | --- |
| `accepted_share` | Meets the session share target but not the network target; never forwarded to consensus |
| `accepted_block` | Meets the network target and the immutable source accepted the candidate |
| `invalid` | Invalid identity/job/lane/share or source-level invalid proof |
| `duplicate` | The current job already saw the same nonce |
| `stale` | The referenced job/work is no longer current |
| `mutated` | The immutable source rejected a template-integrity mismatch |

A network-target candidate is forwarded through `WorkSource.Submit` once. The RPC adapter reconstructs `MiningSolution` from its stored `MiningWorkTemplate` plus the candidate nonce only; candidate job ID, identity, lane and altered work fields cannot rewrite the RPC solution.

## Resource limits

- decoded message: maximum 64 KiB
- full worker identity: maximum 128 bytes
- worker name: maximum 32 bytes
- job ID: maximum 128 bytes
- retained stale job IDs: maximum 8 per session
- duplicate-share keys: maximum/default 65,536 for the current job

Duplicate tracking is reset on clean work. Reaching the configured duplicate limit fails additional submissions closed until a new current job is installed.

## Protocol errors

| Code | Meaning |
| ---: | --- |
| `-32700` | parse error |
| `-32600` | invalid request |
| `-32601` | method not found |
| `-32602` | invalid params |

## Security and deployment boundary

The Stage D package is deliberately transport-independent. `pool/stratum` does not open sockets, construct blocks, persist balances, or modify chain state. The RPC adapter copies the provider block, changes only the validated reward address before issuance, stores the exact returned immutable template and submits only a reconstructed solution based on that template plus nonce.

Stage E and Stage F remain injection-only infrastructure. They do not choose a bind address, call a socket-listening primitive, load certificate/key files, wire themselves into the node or expose a public endpoint. Stage G is the only staged socket owner and is intentionally incapable of selecting a public address: its zero-argument `loopback.Listen()` API opens only `tcp4` at `127.0.0.1:0`, and a source guard rejects configurable or alternate binding paths.

Deferred work includes any public/deployment-specific socket ownership, explicit public endpoint configuration, proxy/IP policy beyond the raw peer address, production authentication, variable difficulty, accounting, payout thresholds, fees, wallet custody, Kryptex-specific extensions, miner packaging, public deployment and any GPU-PoW activation height.

The permanent Stage D offline gate is:

```bash
go test ./pool/stratum ./rpc -run 'Stratum|OfflineStratumTranscript' -count=1 -v
```

Passing these software gates is interoperability evidence only. It is not physical GPU evidence and is not a Kryptex onboarding or listing claim.

## Stage E injected connection transport

`ServeConn` accepts exactly one already-open `net.Conn`, creates exactly one Stage D session, and owns and closes that connection. It does not open a listener or create an accept loop.

The injected transport accepts LF and CRLF framing with a strict 64 KiB request-line bound. Each connection has finite read and write deadlines, cancellation wakes blocked I/O, and all background refresh work is stopped before `ServeConn` returns.

After successful authorization, work is delivered immediately and then refreshed periodically from the immutable Stage D source. Identical work produces no new notification; changed work produces a serialized `mining.set_difficulty` and `mining.notify` pair. All responses and refresh notifications share the same mutex-protected writer.

Each connection has an independent token bucket and a finite recoverable protocol-error budget. Oversized lines fail closed with a best-effort stable invalid-request response. These controls do not change the frozen worker identity, nonce-lane, immutable-job, duplicate-share, stale-share, or block-candidate contracts.

Stage E still has no listener, TLS termination, public endpoint, trusted-proxy or IP admission policy, vardiff, payout/accounting/custody behavior, or Kryptex-specific extension.

The permanent Stage E gate is:

```bash
go test -race ./pool/stratum/... ./rpc -run 'Stratum|Transport|OfflineStratumTranscript' -count=1 -v
```

## Stage F injected listener supervisor

`server.ServeListener` accepts an **already-created** `net.Listener`. It owns that injected listener for the duration of the call, accepts streams, applies listener-level admission, optionally performs TLS termination from a caller-supplied `*tls.Config`, and delegates every admitted stream to Stage E `transport.ServeConn`.

Stage F defaults to a maximum of **256 concurrent connections globally** and **8 concurrent connections per source IP**. For TCP peers the admission key is the canonical remote IP only; source ports are ignored and IPv4-mapped IPv6 addresses normalize to IPv4. Non-TCP addresses fall back to `Addr.String()`. Stage F does not parse PROXY protocol and does not trust reverse-proxy headers, so an operator must not place it behind an address-multiplexing proxy and expect per-client admission accounting without a separately designed trusted-proxy layer.

TLS is optional. Stage F accepts only a caller-supplied TLS configuration, clones it during normalization, requires **TLS 1.2 or newer**, and applies a default **10-second handshake timeout**. Stage F does not read certificate or private-key files itself. A TLS handshake failure or timeout is connection-local and does not create a Stage D session.

Temporary listener accept failures use a bounded default retry backoff of **100 ms**. Permanent accept failures terminate the supervisor with context. Admission rejection, TLS failure, Stage E protocol/rate-limit termination, normal EOF, and other connection-local errors are contained to that connection; admission capacity is released when the connection goroutine exits.

Cancellation closes the injected listener to wake a blocked `Accept`, aborts active raw connections so TLS/Stage E cleanup cannot stall shutdown, and waits for all admitted connection goroutines before `ServeListener` returns. Source-guard tests reject production calls to `net.Listen`, `tls.Listen`, `http.Serve`, `http.ListenAndServe`, and `http.ListenAndServeTLS`, plus production helper declarations beginning with `ListenAndServe`.

Stage F still provides **no bind address**, certificate loading, node wiring, public Stratum endpoint, trusted-proxy support, vardiff, accounting, payouts, fees, custody, persistent miner accounts, or Kryptex approval claim. It does not change GPU-PoW consensus activation or deployment state.

The permanent Stage F gate is:

```bash
go test -race ./pool/stratum/... ./rpc -run 'Stratum|Transport|Server|OfflineStratumTranscript' -count=1 -v
```

## Stage G loopback-only real-socket interoperability

`loopback.Listen()` is a deliberately constrained socket owner for compatibility and test use. It has no arguments and binds exactly one IPv4 loopback listener using `net.Listen("tcp4", "127.0.0.1:0")`. The kernel selects the ephemeral port. Runtime validation rejects any returned address that is not IPv4 loopback or that retains port zero.

A test-only AST source guard requires exactly one production listen call in `pool/stratum/loopback`, requires literal `tcp4` and `127.0.0.1:0` arguments, rejects address-selection helpers/environment/flag sources, rejects alternate listen paths, and requires the exported `Listen` function to remain zero-argument. Stage G therefore cannot be configured into a public endpoint through its current API.

The compatibility suite in `compatibility/stratum` starts Stage F on the real Stage G listener and uses actual OS TCP sockets. The plaintext test proves subscribe, `WALLET.WORKER` authorization with password `x`, immediate difficulty/job delivery, `accepted_share`, `accepted_block`, duplicate rejection, and forwarding of exactly one network candidate. A separate real-socket case proves the blank password compatibility placeholder also authorizes and receives work.

The TLS compatibility test generates its ECDSA P-256 key and self-signed x509 certificate entirely in memory, configures Stage F through the existing caller-supplied TLS boundary, proves a plaintext client is rejected before a Stage D session is created, then proves a trusted test client completes the Stratum flow over real TCP + TLS 1.2 or newer. No key or certificate fixture is written to disk or committed.

Stage G is **local interoperability evidence only**. It is not wired into `cmd/sudharma-rpcd`, does not create a stable/public port, does not define reverse-proxy trust, does not implement vardiff/accounting/payouts/custody, does not deploy to AWS or Seed-1/Seed-2, does not activate GPU-PoW, and does not imply Kryptex approval or listing.

The permanent Stage G gate is:

```bash
go test -race ./pool/stratum/... ./compatibility/stratum ./rpc -run 'Stratum|Transport|Server|Loopback|OfflineStratumTranscript' -count=1 -v
```
