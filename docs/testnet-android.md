# Sudharma Public Testnet 1 — Android access

Sudharma's public testnet is designed so Android wallets and dApps connect to remote full nodes rather than requiring every phone to maintain the full blockchain.

## Mobile connection model

An Android wallet needs only public testnet metadata:

- network name: `Sudharma Public Testnet 1`
- network slug: `sudharma-testnet-1`
- RPC endpoint: `https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com`
- native symbol: `SUDH`
- explorer URL: `/explorer` on the public website (Amplify host until custom domain publish)
- faucet URL: `/faucet` on the public website; API also at `/v1/faucet/*` on the RPC endpoint

The Android application keeps private keys and transaction signing on the device. The remote RPC node receives only signed transactions and public queries. Wallet passwords, recovery secrets and private keys must never be sent to the RPC service.

## Required user flow

1. Create or import a wallet locally on Android.
2. Select **Sudharma Public Testnet 1**.
3. Request test SUDH from the in-wallet faucet or the public `/faucet` web page.
4. Query balance and account nonce over RPC.
5. Build and sign a transaction locally.
6. Submit the signed transaction to RPC.
7. Track it from pending to confirmed.
8. View the confirmed transaction in the public explorer.

Testnet SUDH has no monetary value and must never be represented as mainnet SUDH.

## Full node on Android

A full node on Android is not required for the public testnet user experience. A future experimental/light-node client may be added, but the supported production model is Android wallet/light client -> HTTPS RPC -> public Sudharma full nodes.

## Security boundary

Public RPC should be placed behind HTTPS, rate limiting and an intentionally managed gateway before mobile release. The node itself does not need or accept user private keys.
