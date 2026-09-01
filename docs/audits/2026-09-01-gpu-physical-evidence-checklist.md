# Physical GPU mining evidence checklist — 2026-09-01

Track completion of the physical Khushi GPU gates required before `PhysicalGPUMiningEvidenceComplete` may flip to `true`.

## Required evidence

| Gate | Status | Evidence location |
| --- | --- | --- |
| RTX 2060 vector/memory/benchmark interoperability | **Recorded** | GitHub #24 |
| RTX 2060 independent localhost staging acceptance (`local-staging-gate=accepted`) | **Pending** | Post to #24 using packaged `run-local-staging-gate.ps1` |
| Physical AMD/non-NVIDIA OpenCL GPU with 4 GiB+ dedicated VRAM | **Pending** | Post to #24 |
| Cross-vendor community reproducibility review | **Pending** | Link review notes in evidence record |

## Owner action to close the gate

1. Complete pending runs on physical hardware and attach evidence directories to #24
2. Record references in private evidence vault using the security-review evidence template
3. Open attestation PR flipping `params.PhysicalGPUMiningEvidenceComplete = true` with links to #24 comments only (no secrets)

## Accuracy requirement

Benchmark-only runs do **not** satisfy the independent staging gate. The controlled staging flow must end with `local-staging-gate=accepted`.
