# Sudharma Wallet Android v0.1 Design Specification

## Product goal
Build a simple, trustworthy Android testnet wallet branded **Sudharma Wallet**. The first release enables Sudharma Testnet only, while the internal architecture is intentionally chain-adapter based so BTC, EVM, Solana and other networks can be added later without redesigning the application.

## Product principles
- Non-custodial: private signing material remains on the user's device.
- Simple UX: familiar portfolio/send/receive flow with minimal technical language.
- Sudharma-first branding: official logo/app icon/splash asset will replace the temporary brand mark when supplied.
- Testnet unmistakable: TESTNET is always visible while a test network is active.
- No ads, subscriptions, staking, dApps, fiat purchase or live swaps in v0.1.
- Swap may appear as disabled `Coming later` affordance only.
- Mainnet must remain unavailable until explicitly released.

## Compatibility authority
The existing Go implementation is authoritative for Sudharma v0.1 protocol compatibility:
- P-256 ECDSA keys.
- Public key encoded with the uncompressed elliptic-curve form used by Go `elliptic.Marshal`.
- Address = lowercase hex of the first 20 bytes of SHA-256(publicKey).
- Transaction ID = lowercase hex SHA-256 of `from|to|amount|fee|nonce`.
- Fee = `(amount * 10) / 10000` using integer arithmetic.
- Signature = 64 bytes: 32-byte big-endian R followed by 32-byte big-endian S.
- Signature payload = UTF-8 bytes of the transaction ID, hashed once with SHA-256 by the signing operation.

Cross-language golden vectors must prove Android output is accepted by the Go implementation before Send is considered complete.

## Recovery and account model
The user-facing default is a standard 12-word BIP39 recovery phrase. The architecture supports 24 words later. Because the current Sudharma Go wallet creates random P-256 keys and has no established HD derivation contract, v0.1 MUST NOT silently invent a permanent Sudharma HD path and call it protocol-standard.

For v0.1:
1. New mobile accounts use a versioned mobile derivation profile documented and covered by golden vectors.
2. The seed-to-P-256 scalar conversion uses deterministic rejection sampling from HMAC-SHA256 domain-separated material and never modulo-bias reduction.
3. The stored wallet metadata records derivation profile/version.
4. Import of the existing encrypted Go wallet file/private-key format is a separate compatibility path and does not reinterpret that key as BIP39.
5. Future derivation profiles can coexist without changing addresses belonging to earlier profiles.

The recovery phrase is the independent recovery authority for mobile-derived accounts. Google Sign-In is never the cryptographic owner of funds.

## Security model
- Android Keystore wraps a random local data-encryption key; hardware-backed/StrongBox is preferred when available, with documented fallback.
- Wallet secrets are encrypted at rest using an authenticated cipher.
- A six-digit app PIN is required and biometrics may unlock/authorize through Android BiometricPrompt; PIN remains fallback.
- PIN verification uses a deliberately slow password KDF and rate limiting/backoff. The PIN is not the BIP39 passphrase and is not a recovery mechanism.
- Recovery phrase/private-key views set FLAG_SECURE and never log secrets.
- Secrets are not sent to Sudharma RPC, analytics, crash reports or Google.
- Clipboard use for secrets is avoided. Address copy is allowed.
- Transactions are assembled and signed locally; RPC receives only public data and signed transactions.

## Google account and cloud backup
Google Sign-In is optional. Users can create/use Sudharma Wallet without an online identity.

If cloud backup is enabled:
- backup plaintext is encrypted on-device before upload;
- cloud contains ciphertext plus non-secret format metadata only;
- the backup encryption design must not make possession of the Google account alone sufficient to decrypt wallet funds;
- users retain the 12-word phrase as independent recovery;
- cloud backup can be disabled.

Google integration is not allowed to block the core offline wallet build. If OAuth/client credentials are not configured, the UI exposes the feature as unavailable rather than embedding secrets in the repository.

## Architecture
Android project lives under `mobile/android/` and uses Kotlin.

Boundaries:
- `app`: Android UI, navigation, lifecycle and dependency wiring.
- `core-model`: chain-neutral money/account/activity models.
- `core-security`: secure storage, PIN/biometric gates, sensitive-screen policy.
- `core-recovery`: BIP39 and versioned account derivation profiles.
- `chain-api`: chain adapter interface.
- `chain-sudharma`: Sudharma address, transaction, signature and RPC implementation.

A chain adapter exposes network identity, address validation, balance, fee estimate, unsigned transaction construction, local signing, submission and transaction status. UI code must not contain Sudharma-specific cryptographic logic.

## Network/RPC
Sudharma adapter consumes the existing RPC contract:
- `GET /v1/status`
- `GET /v1/accounts/{address}`
- `POST /v1/transactions`
- `GET /v1/transactions/{txID}`

Testnet RPC base URL is build/configuration data, not a secret. Cleartext HTTP is not permitted for a public production endpoint; test/development exceptions must be explicitly scoped. Mainnet remains disabled.

## User experience
### Cold launch
Short branded splash (target <= 2.5 s): dark premium background, Sudharma brand mark reveal, `SUDHARMA`, `Sudharma Wallet`, then onboarding/unlock. Animation must respect Android reduced-motion/accessibility settings.

### Onboarding
`Create New Wallet` and `Import Wallet` are primary choices. Optional `Continue with Google` is secondary and must explain that Google is not custody/recovery authority.

New-wallet flow: create 12 words -> require explicit backup acknowledgement and word verification -> create 6-digit PIN -> offer biometrics -> portfolio.

### Portfolio
Trustworthy, simple multi-asset layout from day one: total portfolio header, Send/Receive/Swap actions, Assets list, Activity and Settings. v0.1 contains only SUDH. Swap is disabled and labelled Coming later.

### Send
Recipient can be pasted or scanned by QR. Validate address -> amount -> fetch/choose nonce and fee -> confirmation showing recipient/amount/fee/network -> PIN/biometric authorization -> local signing -> submit -> status screen. Never claim success until RPC accepts the transaction.

### Receive
Show SUDH, network, address, QR, Copy and Share. TESTNET must be visually obvious.

### Activity
Show locally known submitted transactions and refresh authoritative status/confirmations from RPC. Clearly distinguish pending, confirmed and failed/not-found states.

### Network selector
Sudharma Testnet enabled. Sudharma Mainnet shown only as unavailable/coming after launch. Architecture supports future chains/networks.

## QR
Receive QR encodes a versioned Sudharma payment URI when possible and always exposes the raw address. Scanner accepts both the supported URI and raw valid address. QR parsing is isolated behind the chain adapter.

## Branding
The official Sudharma logo is required before a branded release candidate. Until supplied, use a clearly temporary vector/text brand mark created in-repo; do not fabricate an 'official' logo. The app name is `Sudharma Wallet` and Android package/application ID uses a Sudharma-owned reverse-domain namespace.

## Testing and release gates
v0.1 is complete only when all applicable gates pass:
- Existing Go CI remains green.
- Android unit tests pass.
- Sudharma cryptographic/address/transaction golden vectors pass in Go and Kotlin.
- Recovery deterministic vectors pass.
- Address/amount/fee/nonce validation tests pass.
- RPC client tests pass against deterministic fixtures/mock server.
- Security tests verify secret persistence is encrypted and sensitive activities/screens request screenshot protection.
- Android lint passes.
- Debug APK builds in CI and is uploaded as a workflow artifact.
- Instrumentation/UI smoke tests run where CI environment supports them; otherwise the limitation is explicitly recorded and PR stays unmerged if the missing test is considered release-blocking.
- No signing key, mnemonic, OAuth secret, keystore or generated APK is committed to Git.
- Secret-safety CI remains green.

## Merge policy
Work occurs on `feature/android-wallet-v0.1` and a PR targets `main`. Merge only when the v0.1 scope is complete and all required checks pass. Any incomplete or failing implementation remains open and must not be merged.

## Deferred
- Live BTC/EVM/Solana adapters.
- Live swap/bridge/exchange integrations.
- Mainnet activation.
- Fiat purchase.
- Staking/dApps.
- Production Play Store signing/publishing.
