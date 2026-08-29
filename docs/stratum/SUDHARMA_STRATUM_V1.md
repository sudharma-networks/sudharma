# Sudharma Stratum V1 Offline Protocol Profile

## Status

This document describes the Stage D **offline interoperability checkpoint** for Sudharma GPU-PoW. It defines the transport-independent Stratum-style message profile implemented by `pool/stratum` and the immutable adapter in `rpc/stratum_adapter.go`.

This checkpoint does **not** expose a public Stratum endpoint, start a TCP or HTTP listener, wire Stratum into `cmd/sudharma-rpcd`, deploy to Seed-1 or Seed-2, activate GPU-PoW, implement pool payouts or custody, or claim Kryptex approval/listing/wire compatibility.

## Wire model

One UTF-8 JSON object is processed per newline-delimited message. Client request IDs may be JSON strings or base-10 integers and are echoed exactly in responses. Batch requests, duplicate object keys, unknown top-level fields, unsupported methods, malformed UTF-8, trailing JSON values and messages above 64 KiB are rejected.

The offline core returns typed protocol errors. A future listener may decide how to frame or close a connection after those errors; no listener policy exists in this checkpoint.

## Client methods

| Method | Parameters | Result |
| --- | --- | --- |
| `mining.subscribe` | `[]` or `[agent]` | 32-character lowercase hexadecimal session ID |
| `mining.authorize` | `[WALLET.WORKER, password]` | `true` after successful binding |
| `mining.submit` | `[WALLET.WORKER, job_id, nonce_hex]` | one submit status string |

For `mining.authorize`, the password is not treated as a credential in this checkpoint. Only an empty string or literal `x` compatibility placeholder is accepted and discarded.

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

Deferred work includes a bounded TCP/TLS listener, connection deadlines, rate limiting, proxy/IP policy, production authentication, variable difficulty, accounting, payout thresholds, fees, wallet custody, Kryptex-specific extensions, miner packaging, public deployment and any GPU-PoW activation height.

The permanent offline gate is:

```bash
go test ./pool/stratum ./rpc -run 'Stratum|OfflineStratumTranscript' -count=1 -v
```

Passing this gate is software interoperability evidence only. It is not physical GPU evidence and is not a Kryptex onboarding or listing claim.
