# Wallet and Demand-Miner Divergence Audit

## Immutable baseline

| Ref | SHA | Role |
|---|---|---|
| `origin/main` | `5c925b47f04aacbbc2e239d0ce1f86357ffa3938` | merged production/testnet foundation |
| `origin/feature/public-testnet-wallet-v2` | `76ae28eaf8d5e9d84ba82302abd471b84304673a` | canonical wallet/faucet integration base |
| `origin/feature/demand-miner-v1` | `bef507cdd37abab6cf02a476137e068c8a024ac0` | isolated demand-miner review line |

Patch-identity comparison reports the demand-miner branch 41 commits ahead and 10 commits behind the canonical base. The ten canonical-only commits include the latest wallet-v2 formatting/CI work, the combined demand-miner integration commit, and the faucet initial-grant reconciliation RED/GREEN pair. They remain authoritative and must not be overwritten.

## Demand-miner hardening classification

| RED | GREEN | Files/area | Integration decision |
|---|---|---|---|
| `e6faee5` | `8429f97` | `demandminer/runner*` | Integrate: stop the unique child only after positive pending-transaction and post-broadcast block evidence. |
| `c47affe` | `e08a48a` | `demandminer/runner*` | Integrate: retain bounded diagnostics while recognizing late evidence through a separate bounded window. |
| `3373310` | `ad997c5` | installer/service | Integrate: preserve shared `/usr/local/bin/sudharmad` and install the reviewed mining child under `/usr/local/libexec/sudharma-demand-miner/sudharmad`. |
| `9de2027` | `baeed07` | service/runtime lock | Integrate: use a systemd-owned runtime directory and stable `/run/sudharma-demand-miner/lock` inode. |
| `a4328b0` | `9ee2b21` | installer | Integrate: reject `DESTDIR + --enable` before any staged mutation. |
| `28a2dd6` | `cd0939f` | installer config target | Integrate: reject symlink and non-regular configuration targets before mutation. |
| `a3f9a98` | `bef507c` | `demandminer/config*` | Integrate: require numeric seed ports in the inclusive range 1–65535. |

Additional unique commits:

- `f76f07d`: integrate deterministic test timing; removes a scheduler-sensitive 250 ms assertion without weakening lifecycle bounds.
- `6d4ad86`: integrate test-only peer admission synchronization through the existing wait helper; no production P2P behavior change.
- `017ba58`: documentation evidence only; retain its activation boundary where consistent with the canonical design.
- Earlier Task 1–6 demand-miner commits are superseded by the canonical combined implementation commit `0ae1c9c`; do not replay them wholesale.
- `7207a61`, `a0b8e37`, and `3babd95`: compare CI assertions after functional hardening; integrate only missing verification, without adding deployment authority.

## Canonical-only work that must remain

- `0b19679` / `fc8ad51`: automatically reconcile a confirmed initial faucet grant before validating a challenge claim.
- `9a7c269` / `c21fc11` / `9830ef3`: canonical Go formatting repair and removal of the one-shot helper.
- `976cfec`: branch CI coverage for wallet-v2.
- `6368c24` / `76ae28e`: removed one-shot diagnostic history; do not restore temporary diagnostics.
- `77c7862`: staging-only capability probe. It grants no Seed-1/Seed-2 deployment or GPU consensus authority.

## Safety boundary

This reconciliation does not deploy or enable the demand miner on Seed-1 or Seed-2. It does not modify GPU/Khushi consensus, activate Mainnet, broaden AWS permissions, add public mining controls, or authorize a live service. Deployment remains a later evidence-gated stage.

## Verification record

- Local Go verification: blocked because the Work runtime has no `go` executable.
- Exact error: `/bin/bash: line 1: go: command not found`.
- Required Go, race, build, rehearsal, container, and formatting checks must therefore be confirmed by GitHub Actions at the exact candidate SHA.
- Demand-miner CI safety assertions: PASS via `bash scripts/check-demand-miner-ci_test.sh`.
- Demand-miner branch CI-source assertions: PASS via `bash scripts/demand-miner-ci_test.sh`.
- Staged installer safety suite: PASS via `bash deployment/testnet/install-demand-miner_test.sh`.
- Lambda dependencies were installed transiently with `npm install --ignore-scripts --no-audit --no-fund`; the generated dependency directory and untracked lockfile were removed from the worktree afterward.
- Lambda suite: 29 tests passed, 0 failed via `npm test`. This includes initial-grant reconciliation, exact confirmed 25 Test SUDH challenge payment, exact 50 Test SUDH reward, replay/cooldown/round guards, uncertain submission, upstream failover, request bounds, and secret-safe logs.
- Android local verification: blocked because `mobile/android/gradlew` is not tracked and no system `gradle` executable is installed. Java is available. Android unit, lint, and APK build gates must run through the exact-head `Android Wallet` GitHub workflow.
- Faucet reconciliation ancestry: both RED `0b19679` and GREEN `fc8ad51` remain ancestors of the integrated candidate.
- GPU/Khushi diff boundary: no changes under `pow`, `params`, `compatibility/cuda`, or `compatibility/opencl` relative to the canonical base.
