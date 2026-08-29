# Sudharma Testnet Public Wallet RPC

This directory contains the public wallet RPC proxy used to expose only the small route surface required by Sudharma Wallet while keeping the seed nodes' administrative RPC private.

## GitHub OIDC trust

The deployment workflow assumes:

- AWS account: `981626123397`
- Role: `Sudharma-GitHub-Actions-Testnet`
- OIDC provider: `token.actions.githubusercontent.com`
- Audience: `sts.amazonaws.com`
- Deployment branch: `feature/android-wallet-v0.1`

GitHub Actions run `33239402641` on 2026-08-29 printed the following safe claims from the actual deployment token:

```text
aud: sts.amazonaws.com
ref: refs/heads/feature/android-wallet-v0.1
repository: sudharma-networks/sudharma
repository_id: 1343485458
repository_owner: sudharma-networks
repository_owner_id: 320107455
sub: repo:sudharma-networks@320107455/sudharma@1343485458:ref:refs/heads/feature/android-wallet-v0.1
```

The AWS role trust policy must therefore match the current `sub`, not the older GitHub subject form that omitted the immutable owner/repository IDs.

Use the following least-privilege trust relationship for this deployment branch:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::981626123397:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
          "token.actions.githubusercontent.com:sub": "repo:sudharma-networks@320107455/sudharma@1343485458:ref:refs/heads/feature/android-wallet-v0.1"
        }
      }
    }
  ]
}
```

Do not widen the subject to arbitrary repositories or branches merely to make deployment succeed.

## Verification after IAM repair

Rerun the `Testnet Public RPC` workflow after updating the AWS trust relationship. A successful deployment must pass all of these gates:

1. tracked-secret safety test;
2. all Lambda proxy tests;
3. GitHub OIDC claim report;
4. AWS OIDC role assumption;
5. Lambda code/configuration update;
6. deployed Lambda configuration verification;
7. HTTPS Function URL lookup;
8. live `GET /v1/status` request through that public URL.

The workflow deliberately does not guess or hardcode the Lambda Function URL. Once the live HTTPS URL passes the final gate, it can be compiled into the Android testnet wallet as its default RPC endpoint so a fresh install requires no manual RPC configuration.

## Public route boundary

The proxy allowlists only wallet-facing routes:

- `GET /health`
- `GET /ready`
- `GET /v1/status`
- `GET /v1/accounts/{address}`
- `POST /v1/transactions`
- `GET /v1/transactions/{transaction_id}`

Administrative, mining, and unrestricted seed-node RPC routes are not exposed by this proxy.
