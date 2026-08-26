# Testnet Wallet Proxy Monitoring and Alarms

Create CloudWatch alarms in `ap-south-1` for `Sudharma-Testnet-Wallet-Proxy` and its HTTP API. These alarms are operational signals; they do not change transaction state.

## Lambda

- **Errors:** Sum >= 5 in 5 minutes.
- **Throttles:** Sum >= 1 in 5 minutes.
- **Duration:** p95 >= 8000 ms for 5 minutes.
- **ConcurrentExecutions:** >= 8 for 5 minutes (reserved concurrency is 10).

## API Gateway HTTP API

- **5xx:** Count >= 5 in 5 minutes, or >= 5% when request volume is high enough for percentage alerting.
- **Latency:** p95 >= 9000 ms for 5 minutes.
- **IntegrationLatency:** p95 >= 8000 ms for 5 minutes.

## Application/degraded state

Lambda logs a structured `wallet_proxy_upstream_unavailable` event when both private seed attempts fail. A log metric filter may count this event and alarm when count >= 3 in 5 minutes.

Do not include signed transaction bodies, recovery phrases, private keys, wallet secrets, authorization headers, AWS credentials, or request bodies in alarm dimensions/log metric filters.

## Operational interpretation

A gateway or Lambda alarm must never cause the Android wallet to mark an uncertain transaction as successful. The client reconciles a deterministic transaction ID through `GET /v1/transactions/{transactionID}`. If both seeds are unavailable, the wallet reports degraded/offline state truthfully.
