# HTTP API Configuration

Create one API Gateway **HTTP API** in `ap-south-1` for the Sudharma public Testnet wallet. Use the AWS-generated `execute-api` hostname; no custom domain is required.

## Integration

- Integration type: Lambda
- Function: `Sudharma-Testnet-Wallet-Proxy`
- Payload format version: 2.0
- Stage: `$default`
- Auto-deploy: enabled

Create only these routes:

| Method | Route |
| --- | --- |
| GET | `/health` |
| GET | `/ready` |
| GET | `/v1/status` |
| GET | `/v1/accounts/{address}` |
| POST | `/v1/transactions` |
| GET | `/v1/transactions/{transactionID}` |

Do not add a `$default` catch-all route. Do not add `/metrics`, `/v1/blocks`, `/v1/mempool`, administrative routes, raw RPC, or Mainnet routes. Lambda independently validates the same allowlist as a second boundary.

## Throttling and limits

For the initial public Testnet stage:

- default steady-state rate: 25 requests/second
- default burst: 50 requests
- transaction submission target: no more than 5 requests/second with burst 10 if route-level throttling is configured
- Lambda request body hard limit: 1 MiB
- Lambda upstream response hard limit: 4 MiB
- Lambda upstream timeout per seed attempt: 3500 ms
- Lambda function timeout: 10 seconds
- Lambda reserved concurrency: 10

API Gateway's platform request limit is not the security boundary; Lambda rejects any transaction request body over 1 MiB before contacting a seed.

## Logging

Enable API Gateway access logging with request ID, route key, status, integration status and latency only. Do not log request or response bodies, authorization headers, signed transaction JSON, wallet addresses as custom log fields, private seed IPs, or Lambda environment values.

Lambda logs route/method/status/latency and request ID but never request bodies or sensitive headers.

## Caching and response behavior

The Lambda forces `Cache-Control: no-store` and `X-Content-Type-Options: nosniff`. Do not put CloudFront/API caching in front of wallet responses at this stage.

## CORS

The Android native wallet does not require browser CORS. Leave CORS disabled unless a separately reviewed browser client is introduced.
