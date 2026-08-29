# Stage G — Loopback Stratum Interoperability Gate Design

## Status

Design for the next software-only Stratum compatibility stage after the completed Stage F injected-listener supervisor.

Stage G proves the full local socket path with real TCP and TLS while preserving the existing prohibition on public exposure, node wiring, GPU-PoW activation and Seed deployment.

## Objective

Add the smallest possible socket-owning boundary that can exercise Stage F over a real operating-system TCP listener without creating a configurable or public endpoint.

The new boundary must:

1. bind only IPv4 loopback;
2. request an ephemeral operating-system-selected port;
3. expose no host, address or port configuration;
4. return an ordinary `net.Listener` that Stage F can supervise;
5. add no protocol semantics of its own;
6. prove plaintext and TLS Stratum conversations over real sockets;
7. leave `cmd/sudharma-rpcd`, Seed-1, Seed-2 and consensus untouched.

## Architecture

### `pool/stratum/loopback`

This package owns exactly one capability: opening a local-only test listener.

Public API:

```go
func Listen() (net.Listener, error)
```

There is intentionally no address argument, port argument, environment-variable lookup or command-line parsing.

`Listen` uses exactly:

```text
tcp4 / 127.0.0.1:0
```

After the operating system returns the listener, the implementation verifies that the returned address is a TCP address, its IP is loopback and its selected port is non-zero. If those invariants do not hold, the listener is closed and the function fails closed.

The package does not accept connections, terminate TLS, create Stratum sessions or know about mining work. Those responsibilities remain in Stage F, Stage E and Stage D respectively.

### Real-socket interoperability tests

A dedicated compatibility test package will create a Stage G loopback listener, run Stage F `server.ServeListener` on it, and connect with a real local TCP client.

The deterministic test source/verifier/lane fixtures are test-only. They do not become production mining behavior.

The plaintext transcript proves:

1. TCP connect to the returned loopback endpoint;
2. `mining.subscribe` response;
3. `mining.authorize` with `WALLET.WORKER` and password `x`;
4. immediate `mining.set_difficulty`;
5. immediate `mining.notify`;
6. extraction of the issued job ID and lane;
7. one known share submission returning `accepted_share`;
8. one known network-target submission returning `accepted_block` and exactly one source submission;
9. duplicate submission returning `duplicate`;
10. clean client close and supervisor cancellation/join.

A second authorization case proves the blank-password compatibility path used by current pool tooling.

The TLS transcript uses the same protocol path through a caller-supplied, in-memory test certificate. It proves a real `tls.Client`/Stage F handshake over the loopback socket, then subscribe/authorize/job delivery. No certificate or private key is committed.

## Kryptex-facing compatibility assumptions

Current Kryptex pool documentation continues to show ordinary Stratum TCP endpoints and `WALLET_ADDRESS.WORKER_NAME` with password `x` or blank for many pools. Some rental-tool instructions also show `/WORKER`; Stage G does not broaden Sudharma's frozen identity grammar on that basis.

The Stage D identity contract remains exactly one dot separator. Any alternative identity syntax requires a separate compatibility decision backed by actual onboarding requirements.

Stage G does not claim Kryptex approval, listing or exact wire compatibility with a specific Kryptex backend.

## Safety properties

### No public bind

Stage G exposes no configurable address. Its production socket owner binds only literal `127.0.0.1:0` using `tcp4`.

A source guard will inspect the production package and fail if:

- the literal loopback endpoint changes;
- an exported listener function gains parameters;
- environment/config/flag-driven address selection is introduced;
- additional socket-listening calls appear.

### No node wiring

Nothing in `cmd/sudharma-rpcd` imports or invokes Stage G. No service config gains a Stratum address or port.

### No consensus or deployment change

The work does not change:

- GPU-PoW activation defaults;
- block validation or difficulty;
- genesis, supply, subsidy or fees;
- miner reward rules;
- Seed-1/Seed-2 services or security groups;
- AWS infrastructure;
- mainnet state.

### No pool business logic

Stage G adds no vardiff, accounting, payout thresholds, fees, balances, custody, authentication database or persistent miner account state.

## Failure handling

`loopback.Listen` wraps operating-system listener errors with context. If post-bind safety validation fails, it closes the listener before returning an error.

All accepted-connection behavior remains Stage F behavior: global/per-source admission, optional TLS handshake bounds, temporary accept retry, connection-local error containment and deterministic shutdown.

## Testing strategy

Stage G is developed test-first on isolated branch `feature/gpu-pow-v1-stage-g`.

Required gates:

1. loopback listener contract tests;
2. source guard for fixed loopback-only socket ownership;
3. real TCP plaintext Stratum transcript;
4. real TCP blank-password authorization case;
5. real TLS Stratum transcript with in-memory certificate;
6. Stage D permanent gate;
7. Stage E permanent race gate;
8. Stage F permanent race gate;
9. new Stage G permanent race gate;
10. activation-default-disabled guard;
11. full repository regression;
12. node build;
13. generic CI including tracked-secret safety, two-node rehearsal, public-testnet container smoke and race detector after integration.

## Integration policy

Stage G is implemented on an isolated branch from the verified Stage F canonical head. It may fast-forward into `feature/gpu-pow-v1` only if the isolated Stage G head is ahead with no divergence and all required GPU-PoW checks pass.

PR #25 stays draft and unmerged. Integration into the feature branch is not authorization to deploy or expose a Stratum endpoint.

## Explicitly deferred

- configurable bind address or fixed public port;
- public DNS endpoint;
- AWS/network load balancer/security-group work;
- certificate/key file loading or ACME;
- trusted proxy / PROXY protocol;
- Kryptex-specific extensions not supported by evidence;
- vardiff;
- pool accounting, payout, fee or custody systems;
- physical RTX 2060 and AMD/OpenCL gates;
- GPU-PoW consensus activation;
- Seed-1/Seed-2 deployment;
- mainnet.
