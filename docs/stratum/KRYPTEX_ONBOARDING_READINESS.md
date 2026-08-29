# Sudharma / Kryptex Pool Onboarding Readiness

**Stage:** H — contract-first external onboarding review

**Last verified against public Kryptex documentation:** 2026-08-29

**Sudharma base checkpoint:** Stage G canonical head `30740efc12fd6a002d2cb5521034c5c5cd3909ca`

## Purpose

This document compares the implemented Sudharma Stratum/GPU-PoW interoperability boundary with currently published Kryptex Pool behavior. Stage H is deliberately documentation-only: it does not widen the Sudharma protocol, add vardiff, expose a public Stratum listener, wire Stratum into `cmd/sudharma-rpcd`, change consensus, deploy to Seed-1/Seed-2, or claim that Kryptex has approved or listed SUDH.

Public Kryptex documentation is evidence of Kryptex's existing pool conventions, not a published SUDH onboarding contract. Any row marked **External confirmation required** must be resolved with Kryptex before Sudharma changes production behavior.

## Public Kryptex references used

- Direct pool mining / account connection (2026-06-04): https://www.kryptex.com/en/articles/kryptex-pools-direct-mining-en
- General pool mining manual: https://pool.kryptex.com/articles/mining-manual-en
- Vardiff and static share difficulty (2026-08-20): https://pool.kryptex.com/articles/vardiff-vs-static-diff-en
- Account-mining overview: https://pool.kryptex.com/articles/account-mining-en

## Readiness matrix

| Area | Published Kryptex behavior | Sudharma Stage D–G behavior | Status | Stage H decision |
| --- | --- | --- | --- | --- |
| Stratum transport | Kryptex publishes ordinary Stratum TCP endpoints and offers SSL/TLS on supported pool paths. | Stage G proves real OS TCP and real TLS over loopback; Stage F terminates caller-supplied TLS 1.2+. | **Software-ready / deployment gated** | Keep transport contract. Do not expose a public endpoint until a separately reviewed deployment layer exists. |
| Subscribe flow | Kryptex miners use normal pool connection flows; exact coin-specific subscribe extensions are not published as one universal contract. | Sudharma supports `mining.subscribe` with zero or one agent parameter and deterministic response IDs. | **External confirmation required** | Ask Kryptex whether SUDH requires any subscribe extensions, extranonce fields, or miner-specific handshake. Do not guess. |
| Authorization password | Published Kryptex examples allow password `x` or blank for direct/account mining. | Sudharma accepts exactly `x` or blank and Stage G tests both over real sockets. | **Ready for current published convention** | Preserve `x`/blank behavior unless Kryptex requests a SUDH-specific extension. |
| Static share difficulty in password | Kryptex documents `d=<number>` for supported algorithm families; algorithm-specific limits/defaults apply. | Sudharma currently rejects passwords other than `x` or blank. Share difficulty is fixed in the session configuration. | **External confirmation required** | Do not add `d=` until Kryptex defines whether Khushi/SUDH uses it and the allowed range/unit. |
| Vardiff | As of 2026-08-20, Kryptex says vardiff is default for SHA256, Scrypt, RandomX, kHeavyHash and Ethash families; PearlHash is an explicit no-vardiff exception. | Sudharma has fixed per-session share difficulty and no vardiff controller. Consensus network difficulty is already separate from share difficulty. | **External confirmation required** | Ask Kryptex to classify Khushi/SUDH as vardiff or static. If vardiff is required, design it as pool-session policy only; never alter consensus difficulty semantics. |
| Worker identity separator | Kryptex general material permits wallet/worker identities with `.` or `/`; account mining commonly uses `mining_username.worker_name`; email paths require `/`. | Sudharma currently requires exactly `WALLET.WORKER`. | **External confirmation required** | Keep canonical SUDH native-wallet `WALLET.WORKER` strict. Do not add slash/email/account usernames until the chosen Kryptex payout/onboarding model is known. |
| Wallet/account identifier | Kryptex account-mining paths can use a Kryptex Mining Username such as `krx...` or email; native coin pool paths may use a payout wallet. | Sudharma `WALLET` is exactly 40 lowercase hexadecimal characters and becomes the reward address in issued work. | **Major contract decision** | Kryptex must confirm whether SUDH onboarding mines to a native SUDH wallet or a Kryptex account identifier. A `krx...`/email value must never be passed into Sudharma's native reward-address field without a separately designed accounting/payout bridge. |
| Worker-name characters | Kryptex public docs commonly describe a worker name up to 32 characters and recommend Latin letters/numbers; examples vary across products. | Sudharma allows 1–32 ASCII characters from `[A-Za-z0-9_-]`. | **Compatible superset, examples should be conservative** | Use alphanumeric worker examples for onboarding until Kryptex validates `_` and `-` for SUDH. No server change required merely for stricter client examples. |
| Job notification shape | Kryptex supports many algorithms/miners; one universal `mining.notify` layout for arbitrary new algorithms is not publicly specified. | Sudharma emits a defined 10-field `mining.notify`: job ID, algorithm, height, target, header prefix, reward address, version, network difficulty, lane, clean-jobs. | **External confirmation required** | Provide Kryptex the full `SUDHARMA_STRATUM_V1.md` profile and ask whether their integration can consume this job shape or requires an adapter/plugin/profile revision. |
| Nonce submission | Coin/algorithm-specific submission formats vary across Stratum implementations. | Sudharma requires `[WALLET.WORKER, job_id, nonce_hex]`, validates a deterministic 32-bit lane in the high nonce bits, rejects duplicate/stale shares, and only forwards network-target candidates. | **External confirmation required** | Kryptex must validate the lane-based 64-bit nonce contract and submission tuple before onboarding. Do not remove lane isolation merely to imitate another coin. |
| Share vs network target | Pool share difficulty is operational and separate from the chain's block target. | Sudharma independently checks share target and immutable network target; accepted shares remain pool-local, block candidates alone reach `WorkSource.Submit`. | **Architecture-ready** | Preserve this separation for any future vardiff/static-difficulty extension. |
| TLS certificates | Kryptex offers SSL/TLS endpoints where supported. | Stage F accepts caller-supplied TLS config; Stage G uses runtime-only test certificates and proves TLS 1.2+ behavior. No certificate-loading/public endpoint exists. | **Software-ready / deployment gated** | Production certificate ownership, DNS, endpoint naming and renewal belong to the future deployment layer. |
| Public endpoint / regions | Kryptex publishes coin-specific regional/global endpoints after support exists. | Sudharma can bind only `127.0.0.1:0` in Stage G and has no public Stratum endpoint. | **Not yet implemented by design** | Do not create a stable/public port until Kryptex wire-profile questions and physical GPU gates are sufficiently resolved. |
| Proxy/source-IP policy | A production pool may sit behind load balancing/proxy infrastructure. | Stage F accounts by raw peer IP and deliberately does not trust PROXY protocol or forwarded headers. | **Deployment design required** | Define trusted-proxy/PROXY-protocol boundaries before any address-multiplexing proxy is placed in front of Stratum. |
| Pool accounting / payouts | Kryptex can credit mining to registered pool wallets/accounts and, on account-mining paths, convert earnings according to its service model. | Sudharma Stage D–G does not persist pool shares, balances, payout thresholds, fees or custody. | **External/product decision required** | Determine whether Kryptex owns accounting/payout completely or needs Sudharma-side pool accounting. Keep accounting outside consensus either way. |
| Miner configuration | Kryptex publishes coin/miner-specific endpoint, worker and password examples after support is defined. | Sudharma has GPU miner packaging/evidence flows, but final Kryptex endpoint/profile examples are intentionally deferred. | **Blocked on external profile** | Publish Windows/Linux/HiveOS-style examples only after endpoint, username, password and job/submit contract are validated. |
| Physical GPU evidence | Kryptex onboarding of a new GPU algorithm needs credible miner/device interoperability evidence in practice even if not described as a universal public checklist. | RTX 2060 CUDA vector/benchmark evidence exists; independent packaged localhost staging remains open. AMD/non-NVIDIA OpenCL 4 GiB+ physical evidence remains open. | **Hardware gates open** | Complete both physical gates and retain reproducible evidence before any consensus deployment decision. |
| Consensus activation | Pool onboarding is separate from chain consensus activation. | GPU-PoW activation remains disabled with no finite activation height; unrestricted GPU mining remains gated. | **Safety gate closed** | Do not activate or deploy GPU-PoW merely to continue Kryptex integration work. |

## Questions to send to Kryptex before changing Sudharma protocol behavior

1. Will SUDH mining be credited to a **native SUDH wallet address** or to a **Kryptex Mining Username/account**?
2. Which exact authorization username grammar should the SUDH endpoint accept: `wallet.worker`, `wallet/worker`, Kryptex username, email, or more than one form?
3. Is password `x`/blank sufficient for SUDH, or must the endpoint also accept `d=<number>`?
4. Should Khushi Algorithm use vardiff? If yes, what initial, minimum and maximum share difficulty values and what target share interval does Kryptex require?
5. Does Kryptex require `mining.set_difficulty` updates during an active connection, and if so when must a new clean job be issued?
6. Can Kryptex consume Sudharma's documented 10-field `mining.notify`, or do they require a different job payload/extranonce model?
7. Can Kryptex consume Sudharma's `[identity, job_id, nonce_hex]` `mining.submit` tuple and 64-bit lane-based nonce model?
8. Are any `mining.configure`, version-rolling, client-reconnect, extranonce-subscribe, keepalive, or other extensions mandatory for the intended miners?
9. Which TCP and TLS endpoint/region conventions should be used after approval, and does Kryptex terminate TLS/load balancing before the Sudharma Stratum service?
10. If a proxy/load balancer is required, what trusted source-IP mechanism is expected (direct source IP, PROXY protocol version, or another mechanism)?
11. Which party owns share accounting, worker statistics, fee policy, payout calculation and custody for SUDH onboarding?
12. Which Windows/Linux/HiveOS miner command formats must Kryptex publish/test for Khushi Algorithm?
13. What minimum physical GPU interoperability evidence does Kryptex want from NVIDIA and AMD/OpenCL before onboarding/testing?
14. What testnet or private interoperability endpoint/process does Kryptex prefer before any public SUDH pool endpoint exists?

## Compatibility changes that are explicitly deferred

The following are **not implementation tasks yet**. They become separate reviewed stages only if the external contract requires them:

- `d=<number>` password parsing;
- dynamic `mining.set_difficulty` / vardiff state;
- slash/email/`krx...` authorization identities;
- accounting or payout ledger;
- PROXY protocol / trusted-proxy source-IP handling;
- public/stable Stratum listener ownership;
- node-daemon Stratum wiring;
- Kryptex-specific miner command templates;
- any consensus activation height or Seed-1/Seed-2 deployment.

## Current go/no-go assessment

**GO for:** Kryptex technical review, wire-profile discussion, offline transcript exchange, private/loopback interoperability development, and physical GPU evidence gathering.

**NO-GO for:** claiming Kryptex listing/support, public pool exposure, accepting Kryptex account identifiers as native SUDH reward addresses, implementing guessed vardiff/static-difficulty rules, GPU-PoW consensus activation, Seed-1/Seed-2 deployment, or mainnet activation.
