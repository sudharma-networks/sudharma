# Stage G Loopback Stratum Interoperability — Completed Implementation Record

**Stage:** G

**Goal:** Add a real OS socket boundary for Sudharma Stratum interoperability while making public exposure impossible through the Stage G API.

**Architecture:** `pool/stratum/loopback` owns one fixed IPv4 loopback listener at `127.0.0.1:0`; Stage F continues to own accept/admission/TLS supervision and Stage E continues to own each admitted connection. Compatibility tests under `compatibility/stratum` speak the existing Stage D protocol over real TCP and TLS.

**Spec:** `docs/superpowers/specs/2026-08-29-stratum-loopback-interoperability-design.md`

## Preserved safety boundary

- [x] Stage G binds only `tcp4` / `127.0.0.1:0`.
- [x] Exported `Listen()` has zero arguments and no host/address/port/configuration input.
- [x] No environment variable, flag or config-file address selection exists.
- [x] No Stratum wiring was added to `cmd/sudharma-rpcd`.
- [x] TLS certificate/key files are not loaded or committed; TLS remains caller-supplied to Stage F.
- [x] Stage D identity/job/share semantics, Stage E framing and Stage F admission/TLS behavior remain unchanged.
- [x] No public endpoint, reverse-proxy trust, vardiff, accounting, payout, fee or custody system was introduced.
- [x] No Seed-1/Seed-2 deployment or AWS change was performed.
- [x] No finite GPU-PoW activation height or unrestricted GPU mining activation was introduced.
- [x] PR #25 remains intended to stay draft/open/unmerged.
- [x] Mainnet remains disabled.

## Task 1 — Fixed loopback socket owner

- [x] Added `feature/gpu-pow-v1-stage-g` to the development workflow trigger.
- [x] Added a real listener contract proving IPv4 loopback, kernel-selected ephemeral port and successful local accept/dial.
- [x] RED proof: GPU-PoW CI run **500** failed at the intentionally missing `Listen` API.
- [x] Implemented only `net.Listen("tcp4", "127.0.0.1:0")` plus runtime address validation.
- [x] GREEN checkpoint: `6cab937c1188ac5c8104b8b79b78418f9244144d`, GPU-PoW CI run **501** PASS.

## Task 2 — Loopback-only source guard

- [x] Added AST tests detecting unsafe public/configurable listener variants.
- [x] RED proof: GPU-PoW CI run **502** failed at the intentionally missing guard helper.
- [x] Implemented guard helpers in test code only.
- [x] Guard requires exactly one production `net.Listen`, literal `tcp4`, literal `127.0.0.1:0`, and zero-argument exported `Listen()`.
- [x] Guard rejects alternate listener creation and address selection from environment/flags/resolution helpers.
- [x] GREEN checkpoint: `65833a96a16a4fd92718a502c739abd16cf288e2`, GPU-PoW CI run **503** PASS.

## Task 3 — Real plaintext TCP interoperability

- [x] Added deterministic test-only Stage D work source, verifier, nonce lane and session factory fixtures.
- [x] Added real OS TCP subscribe -> authorize -> immediate difficulty/job delivery -> share -> block -> duplicate transcript.
- [x] The test extracts the issued job ID from `mining.notify`; it does not reimplement job derivation.
- [x] The test proves `accepted_share` stays pool-local and exactly one network-target candidate reaches `WorkSource.Submit`.
- [x] Added a fresh real-socket case proving blank password authorization and immediate work delivery.
- [x] Run **506** stopped at formatting only; no protocol test executed.
- [x] Corrected only the gofmt alignment in the test fixture.
- [x] GREEN checkpoint: `67c423c0348083f7cb7b0b1766fd106b62e02fe6`, GPU-PoW CI run **507** PASS including full regression and node build.

## Task 4 — Real TLS interoperability

- [x] Added runtime-only ECDSA P-256/self-signed x509 test certificate generation.
- [x] No private key or certificate fixture is written to disk or committed.
- [x] A plaintext client to the TLS-enabled loopback endpoint is rejected before the Stage D session factory is called.
- [x] The same listener remains usable afterward, proving failed TLS admission is released.
- [x] A trusted in-memory TLS client negotiates TLS 1.2+ and completes subscribe/authorize/work/share/block over real TCP + TLS.
- [x] GREEN checkpoint: `193949321eec5736add0a2a8384bd898f2b12e1c`, GPU-PoW CI run **508** PASS including full regression and node build.

## Task 5 — Permanent gate, documentation and integration readiness

- [x] Added permanent `Stage G real loopback Stratum interoperability gate` after Stage F.
- [x] Permanent command uses `-race` over `./pool/stratum/... ./compatibility/stratum ./rpc` with Stage D/E/F/G selectors.
- [x] Updated `docs/stratum/SUDHARMA_STRATUM_V1.md` with the loopback-only socket boundary, real TCP/TLS evidence and deployment exclusions.
- [x] Isolated final implementation/docs head before this record: `54e67682fec5193049216cbd5be568b101555674`.
- [x] Exact-head isolated GPU-PoW CI run **510** PASS, including Stage D, Stage E, Stage F, the new Stage G race gate, activation-default-disabled guard, full regression and node build/checksum.
- [ ] Reconfirm canonical `feature/gpu-pow-v1` ancestry immediately before integration.
- [ ] Fast-forward canonical branch without force.
- [ ] Verify canonical exact head with GPU-PoW and generic CI.
- [ ] Update PR #25 and issue #13 metadata after canonical checks are green.
- [ ] Perform final safety audit.

## Remaining gates after Stage G

Stage G is local interoperability evidence only. Remaining work includes:

- physical RTX 2060 packaged localhost staging round trip and retained evidence bundle;
- AMD/non-NVIDIA OpenCL physical evidence on a GPU with at least 4 GiB dedicated VRAM;
- independent cross-vendor hardware evidence review;
- a separately reviewed public/deployment socket-owner layer before any public pool endpoint can exist;
- explicit proxy/IP operational design before a reverse proxy is introduced;
- validation of the Sudharma profile against Kryptex onboarding requirements and Kryptex-side approval/configuration;
- vardiff only if externally required;
- pool accounting/payout/fee/custody design if pool operation is pursued;
- explicit later consensus activation and controlled Seed-1/Seed-2 deployment criteria;
- mainnet remains a separate future decision.
