# Bounded Stratum Listener Supervisor — Completed Implementation Record

**Stage:** F

**Goal:** Build a bounded injected-listener supervisor above Stage E that enforces aggregate admission, optional TLS and deterministic shutdown without binding a port or exposing Stratum publicly.

**Architecture:** `pool/stratum/server` receives an already-created `net.Listener`, performs listener-level admission and optional TLS handshakes, and delegates every admitted stream to existing `transport.ServeConn`. Stage F never creates a listening socket, never loads certificate files and is not wired into `cmd/sudharma-rpcd`.

**Spec:** `docs/superpowers/specs/2026-08-29-stratum-listener-supervisor-design.md`

## Preserved safety boundary

- [x] No `net.Listen`, `tls.Listen`, `http.Serve`, `http.ListenAndServe` or `http.ListenAndServeTLS` in Stage F production code.
- [x] No Stratum wiring in `cmd/sudharma-rpcd`.
- [x] No TLS certificate/key file loading; only caller-supplied `*tls.Config`.
- [x] No PROXY protocol or reverse-proxy-header trust.
- [x] No vardiff, accounting, payouts, fees, custody or persistent miner accounts.
- [x] No Seed-1/Seed-2 deployment.
- [x] No finite GPU-PoW activation height.
- [x] No unrestricted GPU mining activation.
- [x] PR #25 remains unmerged during Stage F implementation.
- [x] Mainnet remains disabled.
- [x] Stage D worker/job/share semantics and Stage E per-connection framing, limiter, deadlines and refresh behavior remain unchanged.

## Task 1 — Configuration and TLS policy

- [x] Added the isolated Stage F branch to GPU-PoW CI during development.
- [x] Added zero-value/default and invalid-configuration tests.
- [x] Proved RED before implementation; GPU-PoW CI run 472 failed on the intentionally missing configuration API.
- [x] Implemented `Config`, `normalizedConfig`, `normalizeConfig` and `ErrInvalidConfig`.
- [x] Defaults are global 256, per-source 8, TLS handshake 10s, temporary accept backoff 100ms.
- [x] TLS configuration is cloned, caller state is not mutated, and TLS 1.2 is the minimum.
- [x] Exact-head GREEN: GPU-PoW CI run 473.

## Task 2 — Source identity and atomic admission

- [x] Added TCP source-IP normalization tests, including IPv4-mapped IPv6 and non-TCP fallback.
- [x] Added sequential and concurrent admission tests under the race detector.
- [x] Implemented mutex-protected global/per-source reservations with underflow-safe release.
- [x] Same-IP source ports share one admission key; different IPs retain independent capacity.
- [x] Exact-head GREEN: GPU-PoW CI run 475.

## Task 3 — Injected plaintext listener lifecycle

- [x] Added deterministic test-only listener and addressed-connection fixtures.
- [x] Added nil-input, global-cap, per-IP-cap and plaintext Stage E delegation tests.
- [x] Proved RED; GPU-PoW CI run 477 failed because `ServeListener` did not exist.
- [x] Implemented injected-listener ownership, admission-before-session creation, per-connection goroutines, Stage E delegation and connection-local error containment.
- [x] Corrected a test-only closed-pipe assertion exposed by full regression; production behavior was unchanged.
- [x] Exact-head GREEN: GPU-PoW CI run 479.

## Task 4 — TLS boundary and accept-error policy

- [x] Added in-memory self-signed TLS tests; no key/certificate fixture is committed.
- [x] Added successful TLS delegation, plaintext rejection, handshake timeout and session-not-created-on-handshake-failure tests.
- [x] Added temporary accept retry and permanent accept failure tests.
- [x] Proved RED; GPU-PoW CI run 480 failed on the intentionally missing TLS preparation API.
- [x] Implemented caller-supplied TLS termination with a finite handshake deadline and deadline clearing before Stage E handoff.
- [x] Implemented bounded temporary accept retry and contextual permanent failure.
- [x] Added raw-connection cancellation abort so TLS close-notify cannot stall listener shutdown.
- [x] Exact-head GREEN: GPU-PoW CI run 485.

## Task 5 — Shutdown proof, architecture guard and permanent gate

- [x] Added cancellation proof for a blocked `Accept`.
- [x] Added proof that active admitted connections are closed/joined before `ServeListener` returns.
- [x] Shutdown proof passed without requiring new production behavior: GPU-PoW CI run 486.
- [x] Added admission-capacity reuse proofs after normal EOF, Stage E rate-limit termination and TLS failure.
- [x] Added a source guard scanning non-test Stage F production files.
- [x] Source guard rejects `net.Listen`, `tls.Listen`, `http.Serve`, `http.ListenAndServe`, `http.ListenAndServeTLS` and socket-owning helper names beginning with `ListenAndServe`.
- [x] Proved no-bind guard RED; GPU-PoW CI run 487 failed on the intentionally missing matcher.
- [x] No-bind guard GREEN: GPU-PoW CI run 488.
- [x] Admission release paths GREEN: GPU-PoW CI run 489.
- [x] Added permanent `Stage F bounded Stratum listener supervisor gate` immediately after the Stage E gate.
- [x] Updated `docs/stratum/SUDHARMA_STRATUM_V1.md` with Stage F defaults, TLS policy, source-IP policy, failure containment and deployment exclusions.
- [x] Permanent Stage F gate was proven in run 491; that run also exposed a test-only closed-pipe timing assertion in full regression.
- [x] Corrected that assertion by reusing the race-safe connection-closed helper.
- [x] Final isolated Stage F exact-head `5cc2d8d645640bdc964d0f9480b0a1168f7a73d2`: GPU-PoW CI run 492 PASS, including Stage D, Stage E, Stage F, disabled-default guard, full regression and node build.

## Integration

- [x] Rechecked `feature/gpu-pow-v1` immediately before integration: it was still at Stage E head `157b98c46e6883932ebda3318cf71fd001bf9bc6`.
- [x] Compared Stage E head to Stage F final isolated head: Stage F was 24 commits ahead and 0 behind, with the Stage E head as merge base.
- [x] Fast-forwarded `feature/gpu-pow-v1` to `5cc2d8d645640bdc964d0f9480b0a1168f7a73d2` using a non-forced ref update.
- [x] PR #25 was not merged by the integration.

## Canonical feature-branch verification

Canonical verification checkpoint before this documentation-only record update: `b16be3c84f616cbfdd220a65e35985a52ab95547`.

- [x] GPU-PoW v1 CI push run 495: PASS.
- [x] GPU-PoW v1 CI PR run 496: PASS.
- [x] Generic CI run 582: PASS.
- [x] Stage D offline Stratum protocol gate: PASS.
- [x] Stage E bounded Stratum transport gate: PASS.
- [x] Stage F bounded Stratum listener supervisor gate: PASS under the race detector.
- [x] GPU-PoW network activation defaults remain disabled.
- [x] Full repository regression: PASS.
- [x] Node build and checksum step: PASS.
- [x] Generic CI tracked-secret safety, local two-node rehearsal, public-testnet container build/smoke and race detector: PASS.

This commit changes documentation only. It is the final Stage F branch mutation; exact-head CI on this record commit is required before the Stage F checkpoint is reported to PR #25 and issue #13. No further Stage F branch edits should follow a green result.

## Remaining gates after Stage F

Stage F is infrastructure only and does not authorize deployment. Remaining work includes:

- physical RTX 2060 packaged localhost staging round trip and retained evidence bundle;
- AMD/non-NVIDIA OpenCL physical evidence on a GPU with at least 4 GiB dedicated VRAM;
- independent review of cross-vendor hardware evidence;
- a separately reviewed deployment-specific public bind/endpoint layer before any pool service exposure;
- explicit proxy/IP operational design if a reverse proxy is later introduced;
- Kryptex profile validation and Kryptex-side approval/configuration;
- vardiff decision and implementation only if required;
- pool accounting/payout/fee/custody design if pool operation is pursued;
- explicit later consensus activation and Seed-1/Seed-2 deployment criteria;
- mainnet remains a separate future decision.
