# Sudharma Wallet RPC Workflow

Step 59 connects the encrypted `sudharma-wallet` CLI to the production RPC API introduced in Step 58.

## Security boundary

Private keys, wallet passwords, and signing operations stay inside `sudharma-wallet`. The RPC node never receives a password or private key and never signs on behalf of a wallet. Only an already-signed transaction is submitted to `/v1/transactions`.

The default RPC endpoint is `http://127.0.0.1:18545`. A different HTTP/HTTPS endpoint may be supplied as the final command argument.

## Commands

```text
sudharma-wallet create <wallet-file>
sudharma-wallet address <wallet-file>
sudharma-wallet verify <wallet-file>
sudharma-wallet node [rpc-url]
sudharma-wallet balance <wallet-file> [rpc-url]
sudharma-wallet send <wallet-file> <to-address> <amount-sudh> [rpc-url]
sudharma-wallet tx <transaction-id> [rpc-url]
```

`balance` opens the encrypted wallet only to derive/use its public address, then asks the node for confirmed balance and nonce information.

`send` performs this sequence:

1. unlock the encrypted wallet locally;
2. query the sender account from RPC;
3. obtain the next confirmed nonce;
4. parse the SUDH amount exactly in atomic units (no floating-point conversion);
5. calculate the protocol transaction fee and check available balance;
6. construct and sign the transaction locally;
7. submit the signed transaction to RPC;
8. print the transaction ID for later lifecycle tracking.

Amounts support up to the number of decimal places represented by `params.CoinDecimals` (currently eight).

## Transaction lifecycle

`GET /v1/transactions/{transaction-id}` and the wallet `tx` command expose these states:

- `pending`: accepted in the node mempool but not yet included in a block; confirmations are zero.
- `confirmed`: included in the active chain; block height, block hash, and current confirmation count are returned.
- not found: the RPC returns HTTP 404 when the transaction is neither in the mempool nor in the active chain.

Transaction IDs must be canonical 64-character lowercase hexadecimal SHA-256 identifiers.

## Operational notes

The wallet uses bounded HTTP response reads and request timeouts through the reusable `rpc.Client`. RPC errors retain their HTTP status code and server error message so command-line callers can distinguish invalid submissions, missing transactions, and unavailable nodes.

The wallet does not automatically retry transaction submission. This prevents a client timeout from blindly creating repeated submissions; use `sudharma-wallet tx <id>` to inspect the known transaction ID before deciding what to do next.
