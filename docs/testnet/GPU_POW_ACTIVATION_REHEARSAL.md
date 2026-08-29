# GPU-PoW Activation and Rollback Rehearsal

## Does Not Authorize Public Activation

This runbook is only for disposable local data directories. It does not
authorize changing Seed-1 or Seed-2, deploying a binary, enabling unrestricted
mining, selecting a public-testnet activation height, or activating mainnet.
`GPUV1TestnetActivationHeight remains disabled` and
`GPUV1MainnetActivationHeight remains disabled` in committed defaults.

Physical RTX 2060 localhost staging acceptance and independent AMD or other
non-NVIDIA OpenCL 4 GiB+ evidence remain prerequisites for any later public
proposal. Hosted CI is not physical GPU evidence.

## Preparation

1. Use three empty disposable directories: two upgraded nodes and one
   legacy-only observer. Do not copy public Seed data into them.
2. Record the source commit, Go version, operating system, node binary SHA-256,
   abort-tool SHA-256, configuration SHA-256 and GPU verifier vector revision.
3. Confirm both committed network activation constants are disabled.
4. Start both upgraded nodes with activation disabled. Confirm Version 1 at
   both tips and matching `/v1/status` and `/ready` output.
5. Stop the nodes and preserve initial directory snapshots before configuring
   the same future disposable activation height on both upgraded nodes.
6. The chosen height must satisfy the 720-block lead rule in production
   configuration. A focused automated fixture may use a smaller boundary only
   to exercise consensus transitions quickly; it never arms a real node.

## Both Nodes Must Be Stopped

Both upgraded nodes must be stopped before changing or aborting an activation
decision. Confirm the RPC and P2P listeners are closed. The node process holds
an exclusive operating-system data-directory lock; the offline abort tool must
refuse to run if that lock is held. Never remove `.sudharma.lock` as a way to
bypass ownership—the operating-system lock, not file existence, is decisive.

## Disposable Boundary Sequence

1. Restart both upgraded nodes with the identical future height and confirm
   matching persisted activation policy, `armed` phase and next block version.
2. Start the legacy observer and synchronize below the boundary.
3. Accept the last Version 1 block on all three nodes.
4. Reject an early Version 2 block without dispatching another proof algorithm.
5. At the exact boundary, submit a Version 2 block whose nonce has been checked
   independently by the Go GPU-PoW verifier.
6. Confirm both upgraded nodes accept it and the legacy observer stops at the
   last Version 1 block.
7. Reject Version 1 at and after the boundary.
8. Restart both upgraded nodes and replay the same active chain using the same
   immutable policy.
9. Exercise a shallow Version 2 fork within `MaxAutomaticReorgDepth`; never
   construct a Version 1 continuation across the activation boundary.

Stop immediately on a checksum mismatch, policy mismatch, verifier-readiness
failure, unexpected block-version acceptance, observer advancement across the
boundary, evidence-write failure or data-directory lock failure.

## Evidence Manifest

Retain one evidence directory per node containing:

- source commit and binary/configuration SHA-256 values;
- activation record and its SHA-256;
- status/readiness snapshots before arming, while armed and after activation;
- ordered block headers spanning the boundary;
- upgraded-node and legacy-observer logs;
- restart/replay and shallow-reorganization results;
- pre-rehearsal data snapshots; and
- a sorted `SHA256MANIFEST.txt` covering every retained file except the
  manifest itself.

An incomplete evidence directory is a failed rehearsal, not a partial pass.

## Abort Before the Boundary

Abort is permitted only when both upgraded tips remain strictly below the
persisted activation height.

1. Stop both nodes and preserve fresh snapshots.
2. Run the offline command separately for each disposable directory, using a
   new evidence destination and the exact expected height:

   ```text
   sudharma-gpu-activation-abort -data-dir <stopped-node-dir> -evidence-dir <new-evidence-dir> -expected-activation-height <height> -confirm-abort
   ```

3. The command must acquire the data-directory lock, validate the stored chain
   tip and activation record, write a 0600 hash manifest, and atomically move
   the original activation record into the evidence directory. It must never
   delete the only copy.
4. Compare both nodes' abort evidence, restore disabled configuration, restart
   both nodes and confirm Version 1-only status.

Never use manual record deletion as an abort procedure.

## Snapshot Recovery At or After the Boundary

There is no in-chain downgrade at or after the activation height. The offline
abort command must refuse. Stop every participant and retain the failed state
unchanged. Recovery is only by restoring the matching pre-activation snapshots
and disabled disposable configurations for every node. Never reinterpret
Version 2 history as Version 1, continue one node from a different snapshot, or
apply this rehearsal recovery process to a public Seed without a separate
documented incident decision and explicit authorization.
