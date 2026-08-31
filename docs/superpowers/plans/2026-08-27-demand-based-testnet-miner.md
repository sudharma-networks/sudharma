# Demand-Based Public Testnet Miner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deploy a testnet-only supervisor that automatically mines exactly one legitimate PoW block when valid transactions are pending and remains idle when the mempool is empty.

**Architecture:** A focused Go package owns configuration, status validation, polling decisions, child-process execution, cooldown, and backoff. A small command wires that package to the existing RPC client and `sudharmad -mineblocks 1`; a hardened systemd unit supplies single-instance locking and lifecycle management on one seed host.

**Tech Stack:** Go 1.26.6, existing Sudharma RPC and native miner commands, `golang.org/x/sys/unix`, Bash, systemd, GitHub Actions, AWS OIDC/SSM only if already authorized.

**Spec:** `docs/superpowers/specs/2026-08-27-demand-based-testnet-miner-design.md`

## Global Constraints

- Mainnet remains disabled; configuration must require environment `public-testnet`, network `sudharma`, coin `Sudharma`, and symbol `SUDH`.
- Do not change consensus, genesis, block reward, difficulty, transaction, or supply rules.
- Do not add premine, arbitrary minting, balance editing, or privileged confirmation.
- Do not place wallet or faucet private keys, credentials, signing material, or wallet files in the service.
- Do not add a public mining-control endpoint or mining capability to the faucet/proxy.
- Mine exactly one block per invocation and never invoke the miner for an empty mempool.
- Deploy one active supervisor on one public-testnet host initially.
- Reuse `sudharmad -mineblocks 1`; do not create a second block-construction path.
- Every behavior change follows red-green-refactor and every completion claim requires fresh verification.

## File Map

- Create `demandminer/config.go`: strict non-secret supervisor configuration and validation.
- Create `demandminer/config_test.go`: configuration and testnet-only guard tests.
- Create `demandminer/supervisor.go`: status decisions, single sequential loop, cooldown/backoff, and child timeout orchestration.
- Create `demandminer/supervisor_test.go`: deterministic supervisor tests using fake clock/status/miner ports.
- Create `demandminer/runner.go`: bounded `sudharmad` child command construction and process lifecycle.
- Create `demandminer/runner_test.go`: exact argument, timeout, cancellation, and output-bound tests.
- Create `cmd/sudharma-demand-miner/main.go`: config loading, RPC adapter, OS lock, signals, and structured logging.
- Create `cmd/sudharma-demand-miner/main_test.go`: command wiring and lock contention tests.
- Create `deployment/testnet/demand-miner.example.json`: safe example configuration.
- Create `deployment/testnet/sudharma-demand-miner.service`: hardened single-host systemd unit.
- Create `deployment/testnet/install-demand-miner.sh`: idempotent staged installation without service enablement by default.
- Create `deployment/testnet/install-demand-miner_test.sh`: static/idempotency safety tests using a temporary root.
- Modify `deployment/testnet/README.md`: installation, enablement, observation, and rollback runbook.
- Modify `.github/workflows/ci.yml`: build and test the new command and deployment installer.
- Create `.github/workflows/deploy-demand-miner.yml` only after confirming the existing AWS role already has the required SSM access; otherwise deployment remains a documented host command pending explicit permission.

---

### Task 1: Strict Testnet-Only Configuration

**Files:**
- Create: `demandminer/config.go`
- Test: `demandminer/config_test.go`

**Interfaces:**
- Produces: `Config`, `LoadConfig(path string) (Config, error)`, `Config.Validate() error`, and parsed duration helpers.
- Consumes: JSON configuration and no secrets.

- [ ] **Step 1: Write failing configuration tests**

Cover defaults only through explicit example values; reject missing fields, non-loopback status URLs, wrong network/coin names, invalid 40-character reward addresses, non-positive durations, a cooldown shorter than the poll interval, relative binary/data paths, and child timeouts below one second.

```go
func TestConfigValidateAcceptsPublicTestnet(t *testing.T) {
    cfg := validConfig()
    if err := cfg.Validate(); err != nil { t.Fatalf("Validate: %v", err) }
}

func TestConfigRejectsPublicStatusURL(t *testing.T) {
    cfg := validConfig()
    cfg.StatusURL = "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com"
    if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "loopback") {
        t.Fatalf("expected loopback rejection, got %v", err)
    }
}

func TestConfigRejectsMainnetIdentity(t *testing.T) {
    cfg := validConfig()
    cfg.ExpectedCoin = "Sudharma Mainnet"
    if err := cfg.Validate(); err == nil { t.Fatal("expected testnet-only rejection") }
}
```

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./demandminer -run 'TestConfig' -count=1`

Expected: FAIL because package/types do not exist.

- [ ] **Step 3: Implement the minimal strict configuration**

Define:

```go
type Config struct {
    Environment     string `json:"environment"`
    StatusURL       string `json:"status_url"`
    ExpectedNetwork string `json:"expected_network"`
    ExpectedCoin    string `json:"expected_coin"`
    ExpectedSymbol  string `json:"expected_symbol"`
    SeedAddress     string `json:"seed_address"`
    RewardAddress   string `json:"reward_address"`
    MinerBinary     string `json:"miner_binary"`
    DataDirectory   string `json:"data_directory"`
    LockFile        string `json:"lock_file"`
    PollEvery       string `json:"poll_every"`
    Cooldown        string `json:"cooldown"`
    FailureBackoff  string `json:"failure_backoff"`
    ChildTimeout    string `json:"child_timeout"`
}
```

Use `json.Decoder.DisallowUnknownFields`, `url.Parse`, `net.ParseIP`, `filepath.IsAbs`, `time.ParseDuration`, and `^[0-9a-f]{40}$`. Require exact values `public-testnet`, `sudharma`, `Sudharma`, and `SUDH`.

- [ ] **Step 4: Run focused and package tests**

Run: `go test ./demandminer -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add demandminer/config.go demandminer/config_test.go
git commit -m "feat(testnet): validate demand miner configuration"
```

### Task 2: Deterministic Demand Supervisor

**Files:**
- Create: `demandminer/supervisor.go`
- Test: `demandminer/supervisor_test.go`

**Interfaces:**
- Consumes: validated `Config` from Task 1.
- Produces: `StatusSource`, `BlockMiner`, `Sleeper`, `Supervisor`, and `Supervisor.Run(context.Context) error`.

- [ ] **Step 1: Write failing decision and loop tests**

Use fakes, not network mocks:

```go
type Status struct { Network, Coin, Symbol string; Height, IssuedSupply uint64; Mempool int }
type StatusSource interface { Status(context.Context) (Status, error) }
type BlockMiner interface { MineOne(context.Context) error }
type Sleeper interface { Sleep(context.Context, time.Duration) error }
```

Tests must prove: empty mempool never calls `MineOne`; positive mempool calls it once; wrong identity returns a terminal error; status errors use failure backoff; successful mining uses cooldown; a remaining positive mempool permits another block only after cooldown; context cancellation exits without another poll; and only one sequential `MineOne` call can be active.

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./demandminer -run 'TestSupervisor' -count=1`

Expected: FAIL with undefined supervisor interfaces.

- [ ] **Step 3: Implement the minimal supervisor**

Implement a single goroutine loop. Validate every returned status against exact configured identities before reading `Mempool`. For `Mempool == 0`, sleep `PollEvery`; for `Mempool > 0`, call `MineOne`, then sleep `Cooldown`; for recoverable status/miner errors, log through an injected `Logger` and sleep `FailureBackoff`. Return immediately for wrong-network identity or context cancellation.

- [ ] **Step 4: Run focused tests and race detector**

Run: `go test -race ./demandminer -run 'TestSupervisor' -count=1`

Expected: PASS with no race reports.

- [ ] **Step 5: Commit**

```bash
git add demandminer/supervisor.go demandminer/supervisor_test.go
git commit -m "feat(testnet): add demand mining supervisor"
```

### Task 3: Bounded Native Miner Runner

**Files:**
- Create: `demandminer/runner.go`
- Test: `demandminer/runner_test.go`

**Interfaces:**
- Consumes: `Config.MinerBinary`, `SeedAddress`, `RewardAddress`, `DataDirectory`, and `ChildTimeout`.
- Produces: `NativeRunner.MineOne(context.Context) error` satisfying `BlockMiner`.

- [ ] **Step 1: Write failing command and lifecycle tests**

Inject an `ExecCommandContext` function. Assert exact safe arguments:

```go
want := []string{
    "-nodeid", "demand-miner-<unique>",
    "-listen", "127.0.0.1:0",
    "-peer", "3.7.253.229:28444",
    "-datadir", "/var/lib/sudharma-demand-miner/run-<unique>",
    "-mineblocks", "1",
    "-testmineraddress", "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
}
```

Also prove the runner rejects output lacking `Pending Transactions: N`, rejects output lacking `Transactions: N`, caps retained output at 64 KiB, deletes only its unique ephemeral run directory, and terminates on timeout/cancellation.

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./demandminer -run 'TestNativeRunner' -count=1`

Expected: FAIL because `NativeRunner` does not exist.

- [ ] **Step 3: Implement the bounded runner**

Use `exec.CommandContext`, a per-run directory created beneath the validated data root, `-mineblocks 1`, and a unique node ID. Capture bounded combined output. Treat success as exit code zero plus evidence that at least one pending transaction was observed and included. Use `defer os.RemoveAll(runDir)` only after verifying `runDir` is an immediate child of the configured data directory.

- [ ] **Step 4: Run focused tests and full package tests**

Run: `go test -race ./demandminer -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add demandminer/runner.go demandminer/runner_test.go
git commit -m "feat(testnet): run bounded transaction miner"
```

### Task 4: Supervisor Command, Lock, Signals, and Logs

**Files:**
- Create: `cmd/sudharma-demand-miner/main.go`
- Test: `cmd/sudharma-demand-miner/main_test.go`

**Interfaces:**
- Consumes: Tasks 1–3 and `rpc.Client.Status`.
- Produces: executable `sudharma-demand-miner -config <absolute-json-path>`.

- [ ] **Step 1: Write failing wiring and lock tests**

Test that missing `-config` fails, invalid config fails before RPC/miner startup, the RPC adapter maps all identity/height/mempool/supply fields, a second lock acquisition fails, cancellation reaches the supervisor, and JSON logs contain event metadata but no configuration contents.

```go
func TestAcquireLockRejectsSecondProcess(t *testing.T) {
    first, err := acquireLock(t.TempDir() + "/miner.lock")
    if err != nil { t.Fatal(err) }
    defer first.Close()
    if _, err := acquireLock(first.Name()); err == nil { t.Fatal("expected lock contention") }
}
```

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./cmd/sudharma-demand-miner -count=1`

Expected: FAIL because the command package does not exist.

- [ ] **Step 3: Implement minimal command wiring**

Use `unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)`, `signal.NotifyContext` for SIGINT/SIGTERM, existing `rpc.NewClient`, and JSON logging via the existing operations logger pattern. Acquire the lock after config validation and before the first status poll. Never log the full config or environment.

- [ ] **Step 4: Run command tests, race detector, and build**

Run:

```bash
go test -race ./cmd/sudharma-demand-miner ./demandminer -count=1
go build -trimpath -o /tmp/sudharma-demand-miner ./cmd/sudharma-demand-miner
```

Expected: both exit zero.

- [ ] **Step 5: Commit**

```bash
git add cmd/sudharma-demand-miner demandminer
git commit -m "feat(testnet): add automatic demand miner command"
```

### Task 5: Hardened Host Deployment Assets

**Files:**
- Create: `deployment/testnet/demand-miner.example.json`
- Create: `deployment/testnet/sudharma-demand-miner.service`
- Create: `deployment/testnet/install-demand-miner.sh`
- Test: `deployment/testnet/install-demand-miner_test.sh`
- Modify: `deployment/testnet/README.md`

**Interfaces:**
- Consumes: built `sudharma-demand-miner` and `sudharmad` binaries.
- Produces: disabled-by-default, idempotent host installation and explicit enable/rollback commands.

- [ ] **Step 1: Write the failing installer safety test**

Run the installer against `DESTDIR=$(mktemp -d)` with fixture binaries/config. Assert exact file destinations/modes; a dedicated `sudharma-miner` user in the rendered instructions; `127.0.0.1` status URL; `flock`/service single-instance protection; `NoNewPrivileges=true`; `PrivateTmp=true`; `ProtectSystem=strict`; writable access only to `/var/lib/sudharma-demand-miner`; no `systemctl enable --now` unless `--enable` is passed; and rollback never references `/var/lib/sudharma`.

- [ ] **Step 2: Run test and verify RED**

Run: `bash deployment/testnet/install-demand-miner_test.sh`

Expected: FAIL because installer/assets do not exist.

- [ ] **Step 3: Implement deployment assets**

The example config must use:

```json
{
  "environment": "public-testnet",
  "status_url": "http://127.0.0.1:28545",
  "expected_network": "sudharma",
  "expected_coin": "Sudharma",
  "expected_symbol": "SUDH",
  "seed_address": "127.0.0.1:28444",
  "reward_address": "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
  "miner_binary": "/usr/local/bin/sudharmad",
  "data_directory": "/var/lib/sudharma-demand-miner",
  "lock_file": "/run/sudharma-demand-miner.lock",
  "poll_every": "10s",
  "cooldown": "30s",
  "failure_backoff": "30s",
  "child_timeout": "5m"
}
```

Document install-disabled, dry observation, enable, `journalctl`, status checks, and rollback commands. State explicitly that the reward address is public and contains no key material.

- [ ] **Step 4: Run installer, secret, unit, and full repository checks**

Run:

```bash
bash deployment/testnet/install-demand-miner_test.sh
bash scripts/check-tracked-secrets.sh
go test ./... -count=1
go vet ./...
go build -trimpath ./cmd/sudharma-demand-miner ./cmd/sudharmad
git diff --check
```

Expected: all exit zero.

- [ ] **Step 5: Commit**

```bash
git add deployment/testnet .github/workflows/ci.yml
git commit -m "ops(testnet): package demand miner service"
```

### Task 6: CI and Deployment Permission Gate

**Files:**
- Modify: `.github/workflows/ci.yml`
- Conditionally create: `.github/workflows/deploy-demand-miner.yml`

**Interfaces:**
- Consumes: Tasks 1–5 and existing GitHub Actions OIDC role.
- Produces: verified binaries/artifacts and, only with existing authorization, an SSM deployment workflow targeting exactly one seed instance.

- [ ] **Step 1: Add a failing CI source assertion**

Extend the repository CI test script or add a shell assertion that requires `go test -race ./demandminer ./cmd/sudharma-demand-miner`, installer tests, secret scans, and both command builds to appear in CI.

- [ ] **Step 2: Run assertion and verify RED**

Run the new focused shell test.

Expected: FAIL until `.github/workflows/ci.yml` contains every required check.

- [ ] **Step 3: Add CI checks and inspect AWS authority without mutation**

Update CI. Then use read-only AWS/GitHub checks to determine whether `Sudharma-GitHub-Actions-Testnet` can call SSM against the single intended seed instance. Do not broaden IAM. If access is absent, stop before deployment and request the exact new permission from the user.

- [ ] **Step 4: If already authorized, add the exact-target deployment workflow**

The workflow must be manual-only, use OIDC, identify one instance by an immutable configured ID or unique tag plus an exact-one assertion, upload/build verified artifacts, install with service disabled, validate config, and require an explicit `enable_service=true` input. It must never target both seeds and must not delete `/var/lib/sudharma`.

- [ ] **Step 5: Verify CI**

Push the implementation branch and require the full CI workflow to pass. Inspect every failed job rather than rerunning blindly.

- [ ] **Step 6: Commit**

```bash
git add .github/workflows/ci.yml .github/workflows/deploy-demand-miner.yml
git commit -m "ci(testnet): verify demand miner deployment"
```

If the deployment workflow was not authorized, omit its path from `git add` and record the permission blocker in the handoff.

### Task 7: Staged Live Testnet Acceptance and Cleanup

**Files:**
- Modify: `deployment/testnet/README.md` only if observed operations differ from the runbook.
- Temporary workflow files are prohibited unless they include a one-shot trigger and are removed immediately after verification.

**Interfaces:**
- Consumes: verified artifacts and one authorized host deployment path.
- Produces: a running single-host service plus evidence for every live acceptance criterion.

- [ ] **Step 1: Capture immutable baseline**

Record both seeds' height, tip, mempool, issued supply, and peer state. Require matching height/tip and mempool zero before enabling.

- [ ] **Step 2: Install without enabling and validate fail-closed behavior**

Run config validation and one private status poll. Confirm wrong-network fixture/config exits non-zero. Confirm no block-height or supply change.

- [ ] **Step 3: Enable exactly one service**

Run `systemctl enable --now sudharma-demand-miner.service`, verify active state, lock ownership, dedicated user, and clean logs. Confirm Seed-2 has no such enabled service.

- [ ] **Step 4: Submit one controlled faucet request**

Use a fresh test wallet address through the normal public faucet. Record the full transaction ID and recipient address privately in the verification log, not repository source.

- [ ] **Step 5: Verify automatic PoW confirmation**

Without invoking any manual miner, require: mempool becomes positive; supervisor logs one child start; height increases exactly one; both seeds converge; mempool returns zero; recipient balance increases by 100 SUDH; transaction status becomes confirmed; and issued supply increases exactly 5,000,000,000 atomic units.

- [ ] **Step 6: Prove idle behavior**

Observe at least `2 × poll_every + cooldown` (50 seconds with example config). Require unchanged height, tip, and issued supply and no second miner child.

- [ ] **Step 7: Verify logs and rollback readiness**

Scan bounded service logs for prohibited secret patterns and wallet material. Execute `systemctl disable --now` once, verify seed service/data remain healthy, then re-enable only after rollback proof passes.

- [ ] **Step 8: Run final repository and live checks**

Run:

```bash
go test -race ./... -count=1
go vet ./...
bash deployment/testnet/install-demand-miner_test.sh
bash scripts/check-tracked-secrets.sh
git diff --check
```

Then repeat both-seed status checks and record the GitHub Actions run URL and deployed binary SHA-256.

- [ ] **Step 9: Commit any runbook correction**

```bash
git add deployment/testnet/README.md
git commit -m "docs(testnet): record demand miner operations"
```

Skip this commit if no tracked file changed.
