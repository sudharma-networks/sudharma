# Public Testnet Explorer Seed Backport

This directory documents the deliberately narrow rollout path for enabling the read-only Blockchain Explorer API on the two existing Sudharma public-testnet seed nodes.

## Safety boundary

This is **not** a normal branch upgrade.

The live seed baseline documented during the original two-node rollout is:

- source commit: `5bad4479d880e804d78969e47a61d1741f3b3215`
- installed `/usr/local/bin/sudharma-rpcd` SHA-256: `a0cc493b1d0237245ec1528ab8a0d13682719aa32f60afa4f6818da9e9661b63`

The Explorer-only application backport is pinned to:

- source commit: `420dc691e3bde30bb933808d541b77325fc41b7d`

**Do not build a seed binary from `feature/public-testnet-wallet-v2`, `main`, or the moving head of this operations branch for this rollout.** Build exactly the pinned Explorer-only application commit above.

The backport changes only Explorer read-model/RPC files plus route registration. It intentionally makes no changes to:

- `consensus/**`
- `pow/**`
- `params/**`
- `p2p/**`
- mining behavior
- node configuration
- persistence format
- GPU-PoW activation
- Mainnet activation

The public website's 51,000,000,000 SUDH hard-cap information is a separate project-level monetary update. This Explorer backport does **not** migrate the old running seed consensus parameters and must not be used as a vehicle for that migration.

## Gate 1 — AWS managed-node access

Both Seed-1 and Seed-2 must be Systems Manager managed nodes before any remote verification or rollout.

Required EC2 tag on **only** the two public-testnet seed instances:

```text
SudharmaManaged=PublicTestnetSeed
```

Each seed needs an EC2 instance profile that provides `AmazonSSMManagedInstanceCore` (the previously created `Sudharma-SSM-InstanceRole` may be used only if it is an EC2-trusted, attachable instance-profile role).

Attach the caller policy in `github-oidc-ssm-policy.json` to the existing GitHub OIDC role `Sudharma-GitHub-Actions-Testnet`. Its `SendCommand` permission is tag-scoped to the two explicitly tagged testnet seeds.

Do not store AWS access keys, SSH private keys, wallet secrets, or seed phrases in this repository.

## Gate 2 — Read-only live verification

Run `.github/workflows/explorer-seed-readonly-verify.yml` manually.

The workflow must prove **for both seeds** before any deployment:

1. the instance is running and SSM `Online`;
2. `sudharma.service` is active;
3. `/usr/local/bin/sudharma-rpcd` exactly matches historical SHA-256 `a0cc493b...`;
4. `/etc/sudharma/node.json` is present;
5. local `/v1/status` is healthy;
6. local `/v1/explorer/status` is still HTTP 404.

If either installed binary checksum differs, **stop**. Do not deploy the backport until the actual running source/binary is identified and compared.

## Gate 3 — One-seed-at-a-time rollout

Only after Gate 2 is green:

1. build `sudharma-rpcd` from exact commit `420dc691e3bde30bb933808d541b77325fc41b7d`;
2. record and verify the built artifact SHA-256;
3. deploy to Seed-1 only;
4. preserve `/etc/sudharma/node.json` and `/var/lib/sudharma` unchanged;
5. retain the previous binary for rollback;
6. restart `sudharma.service`;
7. require service active, `/ready` healthy, `/v1/status` healthy, and `/v1/explorer/status` HTTP 200;
8. verify the public API still serves normal RPC and now serves Explorer reads;
9. only then repeat the same process for Seed-2.

Any health, peer, height, tip, persistence, or RPC regression requires immediate rollback of that seed and stops the rollout.

## Gate 4 — Website activation

Only after the settled public endpoint returns successful Explorer data through:

```text
https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com/v1/explorer/status
```

may the website Explorer be pointed at the canonical public testnet API.

Until then, the website must keep its honest unavailable state and must not fabricate chain height, blocks, transactions, peer count, or supply data.
