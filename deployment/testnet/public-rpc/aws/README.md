# Sudharma Testnet Public Wallet API — AWS Boundary

This directory documents the owner-controlled AWS setup for the Sudharma Testnet wallet proxy. It contains no AWS credentials, wallet secrets, signing material, private keys, recovery phrases, or API keys.

## Existing resources

- AWS account: `981626123397`
- Region: `ap-south-1`
- VPC: `vpc-0cd862d72cf8165fa`
- Lambda function: `Sudharma-Testnet-Wallet-Proxy`
- Lambda execution role: `arn:aws:iam::981626123397:role/Sudharma-Wallet-Proxy-Lambda-Execution`
- GitHub OIDC deployment role: `arn:aws:iam::981626123397:role/Sudharma-GitHub-Actions-Testnet`
- Lambda security group: `sg-057c9893359ab2300`
- Seed-1 private target: `http://172.31.10.171:29100`
- Seed-2 private target: `http://172.31.32.195:29100`

Raw node RPC on TCP `28545` remains loopback-only and is never an AWS security-group destination.

## Runtime

The Lambda runtime is **Node.js 24.x**, architecture `x86_64`, handler `index.handler`. The code package contains only the tested files under `deployment/testnet/public-rpc/lambda/`.

Recommended runtime settings for Testnet:

- memory: 128 MB
- timeout: 10 seconds
- reserved concurrency: 10
- environment: `SEED_1_URL`, `SEED_2_URL`, `UPSTREAM_TIMEOUT_MS=3500`

The environment values are non-secret network configuration. Do not put credentials or wallet material in Lambda environment variables.

## IAM

`../lambda-execution-policy.json` is the custom Lambda runtime policy. Its EC2 ENI actions require `Resource: "*"` because Lambda VPC network-interface APIs are not fully resource-scopeable; the action set is intentionally limited to VPC ENI lifecycle plus the function's CloudWatch Logs stream.

`../github-actions-testnet-policy.json` is the custom GitHub OIDC policy. It permits only read/update operations against the exact `Sudharma-Testnet-Wallet-Proxy` Lambda ARN. It does not create IAM users, access keys, API Gateway resources, or broad AWS resources.

Initial API Gateway creation and the first IAM-policy attachment are AWS-owner actions. Subsequent Lambda code/config updates can use GitHub OIDC after the custom policy is attached to the existing OIDC role.
