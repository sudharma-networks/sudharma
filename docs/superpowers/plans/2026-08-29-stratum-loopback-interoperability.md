# Stage G Loopback Stratum Interoperability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a loopback-only real-socket boundary and prove the Sudharma Stage D/E/F Stratum stack end to end over actual local TCP and TLS without creating any public endpoint or node wiring.

**Architecture:** A new `pool/stratum/loopback` package owns one fixed `tcp4` listener at `127.0.0.1:0` and returns it as `net.Listener`; Stage F remains the accept/TLS/admission supervisor. A separate compatibility test package constructs deterministic test-only Stage D sessions, starts Stage F on that listener, and speaks Stratum over real local sockets.

**Tech Stack:** Go 1.26.6, standard `net`, `crypto/tls`, `crypto/x509`, existing `pool/stratum`, `pool/stratum/transport`, `pool/stratum/server`, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-08-29-stratum-loopback-interoperability-design.md`

## Global Constraints

- Stage G production binding is exactly `tcp4` / `127.0.0.1:0`.
- The exported socket-owner API has no host/address/port/configuration parameter.
- Stage G does not import or modify `cmd/sudharma-rpcd`.
- Stage G does not load TLS certificate/key files; TLS remains caller-supplied to Stage F.
- Stage G does not change Stage D identity/job/share semantics, Stage E framing or Stage F admission/TLS behavior.
- No public endpoint, Seed-1/Seed-2 deployment, AWS change, GPU-PoW activation, vardiff, accounting, payouts, fees or custody.
- `WALLET.WORKER` remains the canonical identity grammar; password `x` and blank are both exercised.
- PR #25 remains draft and unmerged.

---

### Task 1: Loopback-only socket owner

**Files:**
- Create: `pool/stratum/loopback/listener.go`
- Create: `pool/stratum/loopback/listener_test.go`
- Modify: `.github/workflows/gpu-pow-v1-ci.yml`

**Interfaces:**
- Consumes: Go `net.Listener`.
- Produces: `func Listen() (net.Listener, error)`.

- [ ] **Step 1: add Stage G branch to the GPU-PoW development workflow**

Add `feature/gpu-pow-v1-stage-g` beside the canonical feature branch and Stage F branch in the push trigger so RED/GREEN commits execute the permanent baseline gates.

- [ ] **Step 2: write the failing listener contract test**

Create `listener_test.go` with tests equivalent to:

```go
func TestListenReturnsEphemeralIPv4Loopback(t *testing.T) {
    listener, err := Listen()
    if err != nil { t.Fatal(err) }
    defer listener.Close()

    addr, ok := listener.Addr().(*net.TCPAddr)
    if !ok { t.Fatalf("address type = %T, want *net.TCPAddr", listener.Addr()) }
    if !addr.IP.IsLoopback() { t.Fatalf("listener IP = %s, want loopback", addr.IP) }
    if addr.IP.To4() == nil { t.Fatalf("listener IP = %s, want IPv4", addr.IP) }
    if addr.Port == 0 { t.Fatal("listener retained ephemeral port zero") }
}
```

Also connect with `net.DialTimeout("tcp4", listener.Addr().String(), time.Second)` and verify `Accept` succeeds so the returned endpoint is a real OS TCP listener.

- [ ] **Step 3: run CI and prove RED**

Expected: formatting succeeds and vet/test compilation fails because `Listen` does not exist.

- [ ] **Step 4: implement the minimal fixed loopback listener**

`listener.go` must use only:

```go
const address = "127.0.0.1:0"

func Listen() (net.Listener, error) {
    listener, err := net.Listen("tcp4", address)
    if err != nil {
        return nil, fmt.Errorf("listen on Stratum loopback: %w", err)
    }
    addr, ok := listener.Addr().(*net.TCPAddr)
    if !ok || addr.IP == nil || !addr.IP.IsLoopback() || addr.IP.To4() == nil || addr.Port == 0 {
        _ = listener.Close()
        return nil, errors.New("unsafe Stratum loopback listener address")
    }
    return listener, nil
}
```

No arguments, environment lookup, flags, config files or alternate addresses.

- [ ] **Step 5: run focused and full GPU-PoW CI to GREEN**

Focused target:

```bash
go test -race ./pool/stratum/loopback -count=1 -v
```

Then require the existing workflow to pass vet, Stage D/E/F, activation-disabled guard, full regression and node build.

- [ ] **Step 6: commit the GREEN checkpoint**

Commit message: `feat(pool): add fixed Stratum loopback listener`

---

### Task 2: Source guard for loopback-only ownership

**Files:**
- Create: `pool/stratum/loopback/source_guard_test.go`

**Interfaces:**
- Consumes: production Go files in `pool/stratum/loopback`.
- Produces: a permanent test proving the socket owner cannot drift toward configurable/public binding.

- [ ] **Step 1: write a failing guard-matcher test**

The test helper must parse a sample unsafe file and detect any of:

```text
net.Listen
net.ListenConfig.Listen
```

It must also identify address-selection calls such as `os.Getenv`, `flag.String`, `flag.StringVar` and `net.ResolveTCPAddr` if introduced in production.

The real package scan must require exactly one production `net.Listen` call whose network literal is `tcp4` and address literal is `127.0.0.1:0`.

- [ ] **Step 2: prove RED**

Expected: test compilation fails because the guard helpers do not yet exist.

- [ ] **Step 3: implement AST guard helpers in the test file only**

Use `go/parser`, `go/ast`, `go/token`, `os.ReadDir`, `filepath` and `strings`. Scan only non-test `.go` files. Fail on configurable address sources, extra listen calls, non-literal arguments, wrong network or wrong address.

Additionally inspect exported functions and require `Listen` to have zero parameters.

- [ ] **Step 4: run focused race tests and full workflow**

```bash
go test -race ./pool/stratum/loopback -count=1 -v
```

Expected: PASS and existing Stage D/E/F gates remain PASS.

- [ ] **Step 5: commit**

Commit message: `test(pool): guard Stage G loopback-only binding`

---

### Task 3: Real TCP plaintext interoperability transcript

**Files:**
- Create: `compatibility/stratum/loopback_test.go`
- Create: `compatibility/stratum/fixtures_test.go`

**Interfaces:**
- Consumes:
  - `loopback.Listen() (net.Listener, error)`
  - `server.ServeListener(context.Context, net.Listener, transport.SessionFactory, transport.Config, server.Config) error`
  - `stratum.NewSession(io.Reader, stratum.WorkSource, stratum.ShareVerifier, stratum.Config) (*stratum.Session, error)`
- Produces: end-to-end real-socket interoperability proof.

- [ ] **Step 1: create deterministic test-only fixtures**

Use a fixed wallet:

```text
9ccdc094489874bed888ffe4bdf9b8298f4c5131
```

Use fixed work:

```go
stratum.Work{
    WorkID: "loopback-work-1",
    Algorithm: "sudharma-gpupow-v1",
    TargetHex: "0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f",
    HeaderPrefixHex: "aabbccdd",
    RewardAddress: wallet,
    Version: 2,
    Height: 7600,
    Difficulty: 11,
}
```

Provide a fixed lane `0x01020304`. The verifier returns true for the share target and returns true for the network target only when the low 32 nonce bits equal `2`. The source records submitted candidates and returns `stratum.SourceAccepted`.

Each session uses deterministic 16-byte entropy so the transcript is reproducible.

- [ ] **Step 2: write the real plaintext TCP test**

Start Stage F in a goroutine on a Stage G listener, then dial `tcp4` to the actual listener address. Set a finite client deadline.

Send newline-delimited requests in this order:

```json
{"id":1,"method":"mining.subscribe","params":["khushi-loopback/1.0"]}
{"id":2,"method":"mining.authorize","params":["9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01","x"]}
```

Decode lines as generic JSON maps and verify:

- subscribe result is a 32-character lowercase hex session ID;
- authorize result is true;
- next method is `mining.set_difficulty` with difficulty 4;
- next method is `mining.notify` and contains algorithm `sudharma-gpupow-v1`, height 7600, the fixed target/header/reward address, version 2, network difficulty 11, lane `0x01020304`, clean-jobs true.

Extract the issued job ID from `mining.notify` rather than reimplementing job derivation in the compatibility test.

Submit:

```text
lane<<32 | 1  -> accepted_share
lane<<32 | 2  -> accepted_block
same block nonce again -> duplicate
```

Verify the source saw exactly one network candidate with the block nonce.

Close the client, cancel the context and require Stage F to return `context.Canceled` within one second.

- [ ] **Step 3: run the focused test**

```bash
go test -race ./compatibility/stratum -run 'Loopback.*Plaintext' -count=1 -v
```

The test may pass immediately because Tasks 1 plus Stages D/E/F already provide the required behavior. If it fails, investigate root cause before changing production code.

- [ ] **Step 4: add blank-password real-socket case**

Use a fresh connection/session and authorize `wallet.rig_blank` with `""`. Require true authorization and immediate work delivery. Do not broaden worker syntax.

- [ ] **Step 5: verify all related gates**

```bash
go test -race ./compatibility/stratum ./pool/stratum/... ./rpc -run 'Stratum|Transport|Server|Loopback|OfflineStratumTranscript' -count=1 -v
```

Expected: PASS.

- [ ] **Step 6: commit**

Commit message: `test(pool): prove real TCP Stratum loopback flow`

---

### Task 4: Real TLS loopback interoperability

**Files:**
- Modify: `compatibility/stratum/fixtures_test.go`
- Create: `compatibility/stratum/tls_loopback_test.go`

**Interfaces:**
- Consumes Stage G listener and Stage F caller-supplied TLS configuration.
- Produces real OS TCP + TLS + Stratum interoperability proof.

- [ ] **Step 1: add an in-memory TLS fixture**

Generate an ECDSA P-256 key and self-signed x509 certificate at test runtime. Use DNS/IP SANs suitable for localhost testing. Never write key material to disk or commit a fixture.

Server config is passed through:

```go
server.Config{
    TLSConfig: &tls.Config{
        Certificates: []tls.Certificate{certificate},
        MinVersion: tls.VersionTLS12,
    },
    TLSHandshakeTimeout: time.Second,
}
```

- [ ] **Step 2: write real TLS transcript test**

Open the Stage G listener, start Stage F with TLS enabled, then dial the actual endpoint with `tls.DialWithDialer` or `tls.Client` over a real TCP connection. The test client may use an in-memory root pool or `InsecureSkipVerify` only inside the test fixture; production code must not change.

Perform subscribe, authorize with password `x`, and verify immediate `mining.set_difficulty` plus `mining.notify`.

- [ ] **Step 3: prove plaintext cannot enter a TLS-enabled Stage F listener**

Open a separate real TCP connection to the TLS-enabled listener, write a plaintext Stratum line, and require the connection to close without creating a Stage D session.

- [ ] **Step 4: run focused race tests and full workflow**

```bash
go test -race ./compatibility/stratum -run 'Loopback.*TLS' -count=1 -v
go test -race ./pool/stratum/... ./compatibility/stratum ./rpc -run 'Stratum|Transport|Server|Loopback|OfflineStratumTranscript' -count=1 -v
```

Expected: PASS.

- [ ] **Step 5: commit**

Commit message: `test(pool): prove real TLS Stratum loopback flow`

---

### Task 5: Permanent Stage G gate, documentation and integration

**Files:**
- Modify: `.github/workflows/gpu-pow-v1-ci.yml`
- Modify: `docs/stratum/SUDHARMA_STRATUM_V1.md`
- Modify: `docs/superpowers/plans/2026-08-29-stratum-loopback-interoperability.md`
- Metadata only after final verification: PR #25 and issue #13

**Interfaces:**
- Produces the permanent Stage G regression gate and canonical checkpoint.

- [ ] **Step 1: add the permanent Stage G CI gate after Stage F**

```yaml
- name: Stage G real loopback Stratum interoperability gate
  run: go test -race ./pool/stratum/... ./compatibility/stratum ./rpc -run 'Stratum|Transport|Server|Loopback|OfflineStratumTranscript' -count=1 -v
```

Do not remove or weaken the Stage D/E/F gates.

- [ ] **Step 2: update the protocol/operator profile**

Document:

- Stage G is loopback-only `tcp4 127.0.0.1:0`;
- it owns socket creation but has no configurable address;
- real TCP plaintext and real TLS paths are tested;
- `WALLET.WORKER` with `x` and blank password are exercised;
- Stage G remains local interoperability evidence only;
- no public endpoint, node wiring, proxy trust, Kryptex approval, vardiff or payout/accounting is implied.

- [ ] **Step 3: run isolated Stage G exact-head verification**

Require GPU-PoW CI to pass formatting, vet, Stage D, Stage E, Stage F, Stage G under race detector, activation-default-disabled guard, full repository regression and node build.

- [ ] **Step 4: compare isolated branch against canonical feature branch**

Canonical base must be the verified Stage F final head. Require Stage G status `ahead`, `behind_by=0`, and matching merge base. If the canonical branch has moved independently, do not force-update; investigate and reconcile first.

- [ ] **Step 5: fast-forward `feature/gpu-pow-v1` without force**

Use a non-forced ref update only after Step 4 is satisfied. Do not merge PR #25.

- [ ] **Step 6: verify canonical exact head**

Require both GPU-PoW and generic CI on the canonical Stage G head. Generic CI must pass tracked-secret safety, full tests, local two-node rehearsal, public-testnet container build/smoke and race detector.

- [ ] **Step 7: update PR #25 and issue #13 metadata**

Record final Stage G SHA/run numbers and mark the loopback socket-owner/interoperability harness items complete. Keep public endpoint, proxy policy, physical GPU evidence, Kryptex approval, vardiff/accounting and activation/deployment items open.

- [ ] **Step 8: final safety audit**

Verify:

```text
PR #25: draft/open/unmerged
GPU-PoW activation: disabled
finite activation height: absent
gpu unrestricted mining: gated
Seed-1/Seed-2: unchanged
mainnet: disabled
Stage G bind: only 127.0.0.1:0 tcp4
public Stratum endpoint: none
```

Commit message for the final branch record: `test(pool): gate real Stratum loopback interoperability`.
