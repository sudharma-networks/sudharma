KHUSHI HARDWARE TEST v0.2.0
===========================

This package is for physical GPU interoperability evidence for the Sudharma Khushi Algorithm (protocol id: sudharma-gpupow-v1).

HOW TO RUN
----------
1. Extract the complete ZIP to a normal Windows folder. Do not run files directly from inside the ZIP.
2. Double-click Run-GPU-Test.bat.
3. Select NVIDIA CUDA or OpenCL.
4. Select the GPU device index. Press Enter to use device 0.
5. Leave the test running until it finishes. The formal evidence run performs a full 60-second selected-profile benchmark after bounded autotuning.

WHAT THE TEST DOES
------------------
- records Windows, GPU, driver/runtime, VRAM and device information;
- rejects CPU fallback;
- checks the canonical Khushi GPU vector;
- checks the production memory policy, including the 2 GiB dataset allocation path and required reserve;
- checks frozen production boundary vectors;
- autotunes bounded launch profiles and measures the selected profile for 60 seconds;
- starts the same-revision Go verifier on 127.0.0.1:28646 only;
- solves one controlled staging challenge on the selected physical GPU;
- submits the nonce to the independent local verifier;
- writes a tamper-evident evidence directory and SHA256 manifest.

SUCCESS
-------
The required final marker is:

local-staging-gate=accepted

Keep the complete khushi-staging-evidence-* directory. It is the retained evidence bundle used for review.

SAFETY BOUNDARY
---------------
This hardware test does not activate network mining or mainnet mining, does not create a block, and does not change consensus activation. It does not contact Seed-1, Seed-2, or a public mining endpoint. The verifier is localhost-only.

A successful software package or local run does not by itself set the project PhysicalGPUMiningEvidenceComplete gate. The retained physical evidence must still be reviewed. The formal cross-vendor gate also requires a real AMD or other non-NVIDIA OpenCL GPU with at least 4 GiB dedicated VRAM in addition to NVIDIA evidence.

RELEASE PACKAGING
-----------------
Khushi Hardware Test v0.2.0 is published as an exact-revision prerelease package. The package records its source revision and SHA256 checksums in retained release metadata.
