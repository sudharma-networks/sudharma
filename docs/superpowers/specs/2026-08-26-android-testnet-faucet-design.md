# Sudharma Android Testnet Faucet Design

Date: 2026-08-26
Status: Proposed for implementation after user review
Target branch: feature/android-wallet-branding-v0.1-finalwork

## Goal

Make the Sudharma Android testnet wallet immediately useful for public testing without requiring testers to enter RPC settings or contact an administrator for test coins.

A fresh wallet will ship with the public Sudharma Testnet RPC already configured and will expose an explicit **Get 100 Test SUDH** action. The action will request a one-time 100 SUDH grant from a dedicated faucet service. Test SUDH has no monetary value and exists only for development/testing.

## Existing proven foundation

The following pieces are already operating and are treated as existing dependencies rather than redesigned components:

- Sudharma Seed-1 and Seed-2 are connected over P2P.
- PoW block production and block rewards are functioning on the public testnet.
- The public wallet RPC path is reachable through AWS API Gateway and the Lambda wallet proxy.
- The public wallet RPC base URL is:
  `https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com`
- Android can read a real on-chain wallet balance through that public RPC.
- The current Android wallet supports local key derivation, Receive, Send, balance lookup, transaction submission, and local transaction tracking.

## User experience

### Fresh install

1. User installs the public testnet APK.
2. Wallet creates or imports an account as today.
3. No RPC setup screen is required for normal use because the public HTTPS RPC URL is the built-in default.
4. The home screen clearly identifies the network as **Sudharma Testnet**.
5. A new action labeled **Get 100 Test SUDH** is available while the address has not yet received a faucet grant.

### Faucet request

1. User taps **Get 100 Test SUDH**.
2. App sends only the wallet's public Sudharma address to the faucet API.
3. Faucet validates address format and checks grant eligibility.
4. If eligible, faucet sends exactly 100 SUDH from a dedicated pre-funded faucet wallet.
5. Faucet returns a transaction identifier/status.
6. App shows a pending/success/failure result and refreshes balance.
7. Once the grant is observed, the action becomes unavailable or reports that the address has already been funded.

The app never sends recovery phrases, private keys, PINs, contacts, or other wallet secrets to the faucet.

## Architecture

### 1. Android wallet

Responsibilities:

- Use the public HTTPS RPC endpoint as the default RPC value.
- Preserve the existing RPC override mechanism for development/troubleshooting.
- Add a small FaucetClient responsible only for requesting test funds.
- Add repository/use-case methods for faucet eligibility/request.
- Add UI state for idle, requesting, funded, already-funded, and error states.
- Refresh on-chain balance after a successful faucet request.

Suggested endpoint:

`POST /v1/faucet/request`

Request body:

```json
{
  "address": "<40-hex-character-sudharma-address>"
}
```

Example success response:

```json
{
  "status": "submitted",
  "amount_atomic": 10000000000,
  "tx_id": "<64-hex-transaction-id>"
}
```

Example duplicate response:

```json
{
  "status": "already_funded"
}
```

### 2. Faucet API

The faucet is a dedicated service, separate from the public read/write RPC proxy.

Responsibilities:

- Accept only faucet-request operations.
- Strictly validate address syntax.
- Enforce one 100 SUDH grant per wallet address for the initial release.
- Enforce API-level rate limiting to limit abuse and accidental request storms.
- Submit a real signed Sudharma transaction from the faucet wallet.
- Record faucet grant state before/atomically with submission so duplicate requests cannot race.
- Return safe public transaction metadata only.

The faucet API must not expose generic signing or arbitrary-send capabilities.

### 3. Faucet wallet

Use a dedicated testnet wallet funded separately from normal wallets.

Properties:

- Holds only testnet SUDH.
- Used exclusively for public faucet grants.
- Private key never ships in the APK or repository.
- Secret material is stored server-side using AWS-managed secret storage/encryption.
- Faucet service receives only the minimum permission required to access this secret.

The faucet is pre-funded by mining an administrator-controlled pool to the faucet wallet. Public faucet requests do **not** trigger mining.

### 4. Grant ledger / idempotency store

A small persistent table records grants keyed by normalized wallet address.

Minimum fields:

- `address`
- `status` (`reserved`, `submitted`, `confirmed`, `failed`)
- `amount_atomic`
- `tx_id` when available
- timestamps

A conditional write on the wallet address prevents two concurrent requests from receiving multiple grants.

For the initial release, eligibility is one grant per address. An administrator-only reset can be designed later; it is not part of the public API.

### 5. Network submission and confirmation

The faucet service should submit transactions through a trusted/private node path where possible rather than granting it broad public administrative access.

The flow is:

Faucet API -> validate/idempotency -> sign fixed 100 SUDH transaction -> submit to Sudharma node -> record tx id -> return submitted status.

Confirmation remains normal Sudharma consensus behavior. The faucet itself does not bypass the mempool or consensus rules.

## Monitoring and observability

The public testnet can be monitored using network-level and faucet-level metadata without collecting wallet secrets.

Recommended metrics/logs:

- Seed node readiness and height
- peer count
- mempool size
- issued supply
- blocks produced
- transaction submission success/failure
- faucet request count
- faucet accepted/rejected/duplicate count
- faucet wallet balance
- faucet transaction ids and confirmation state
- API latency/error rate

Do not log recovery phrases, private keys, PINs, or signed-secret material.

## Abuse controls

Initial controls should stay simple:

- one 100 SUDH grant per normalized wallet address
- API Gateway throttling/rate limits
- request-size limits
- strict JSON schema/address validation
- idempotency/conditional database writes
- no arbitrary amount supplied by clients
- no arbitrary source wallet supplied by clients

IP/device identifiers are not required for the first version. If address-only protection proves insufficient, stronger controls can be designed later with a separate privacy review.

## Security boundaries

- Android wallet signs ordinary user transactions locally.
- Faucet signing happens server-side using only the dedicated faucet key.
- Public RPC Lambda and faucet service are distinct responsibilities.
- Faucet cannot mine blocks.
- Faucet cannot spend from user wallets.
- Clients cannot choose faucet amount.
- Clients never receive faucet private-key material.

## Public documentation

Add a public testnet-testing document in the repository explaining:

- this is a public test network
- test SUDH has no monetary value
- APK install steps
- built-in RPC behavior
- how to use Get 100 Test SUDH
- how to test Send and Receive between two wallets
- how to report problems through GitHub Issues
- never share a recovery phrase/private key in a GitHub issue

## Failure handling

Android:

- network timeout -> show retryable error
- already funded -> explain grant is one-time for that address
- faucet depleted -> show temporary-unavailable message
- transaction submission failure -> show error and allow safe retry depending on server status

Server:

- invalid address -> 400
- duplicate address -> deterministic already-funded response
- rate limit -> 429
- faucet balance insufficient -> 503
- node/RPC unavailable -> 503
- unexpected internal error -> 500 with no secret data

Grant reservation and submission need an explicit recovery strategy so a transient node failure does not permanently consume an address's grant. A failed reservation may be retried by the service, or transitioned to a retryable failed state after reconciliation.

## Testing strategy

### Android unit tests

- default RPC equals the public HTTPS endpoint
- custom override still works
- faucet request serialization/parsing
- faucet UI/repository states
- duplicate/already-funded behavior
- no faucet request includes private wallet material

### Faucet unit tests

- valid/invalid addresses
- fixed grant amount is always 100 SUDH
- duplicate address idempotency
- concurrent duplicate requests
- depleted faucet handling
- node submission errors
- retry/reconciliation behavior

### Integration tests

- mocked node transaction submission
- end-to-end API Gateway -> faucet service request
- one address receives exactly one grant
- second request returns already-funded

### Public testnet validation

Before release, use two fresh Android wallets:

1. wallet A requests 100 SUDH
2. wallet A balance becomes 100 SUDH
3. wallet A sends a small amount to wallet B
4. transaction appears in mempool
5. transaction is mined/confirmed using normal PoW
6. both wallet balances update correctly
7. wallet A cannot claim the faucet grant a second time

## Rollout order

1. Permanently configure the Android default RPC endpoint.
2. Validate ordinary Phone A -> Phone B send/receive on the current testnet before exposing the faucet publicly.
3. Create and pre-fund the dedicated faucet wallet.
4. Implement the faucet API, idempotency store, and rate limits.
5. Connect Android Get 100 Test SUDH action.
6. Add public testnet documentation.
7. Run full Android and faucet tests.
8. Perform public-testnet end-to-end validation.
9. Publish the updated test APK only after all above checks pass.

## Non-goals for this version

- mainnet faucet support
- automatic mining per request
- GitHub account requirement
- email/login requirement
- faucet amounts other than 100 SUDH
- repeated grants/cooldown system
- user analytics or device tracking
- custodial storage of user wallet keys

## Success criteria

The feature is complete only when a fresh Android wallet can use the built-in RPC, explicitly request 100 real on-chain test SUDH without manual administrator intervention, receive the grant at most once per address, and then successfully participate in normal Sudharma send/receive transactions under consensus rules.
