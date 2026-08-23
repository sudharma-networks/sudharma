# Sudharma Wallet Safety and Architecture

## Current production wallet scope

The Sudharma CLI wallet keeps private keys local in encrypted wallet files. The RPC node never receives wallet passwords or private keys; it only receives already-signed transactions.

Core wallet safety commands:

```text
sudharma-wallet create <wallet-file>
sudharma-wallet verify <wallet-file>
sudharma-wallet backup <wallet-file> <backup-file>
sudharma-wallet passwd <wallet-file>
```

`backup` first decrypts/verifies the source wallet, creates a new backup without overwriting an existing file, and then reloads the backup to prove that it resolves to the same public address.

`passwd` verifies the existing password, writes and verifies a newly encrypted copy, preserves the original during replacement, and only removes the recovery copy after the new wallet has been re-opened successfully. Password rotation does not change the wallet address or private key.

## Owner and user safety rules

- Never commit encrypted wallet files, passwords, private keys, recovery secrets, or seed phrases to GitHub.
- Store at least two verified offline backups in physically separate locations before putting material value on an address.
- A public address is safe to share. A private key, wallet password, encrypted wallet file, or future recovery phrase is not.
- Do not assume that knowing a wallet password alone is sufficient recovery; the encrypted wallet file (or a future standard recovery phrase) is also required.
- Verify restored backups by confirming the resulting public address before relying on them.

## Multi-chain / universal-wallet direction

Sudharma is the primary/default network, but the wallet should evolve into a modular multi-chain wallet rather than hard-code all future user experience around one chain.

The intended architecture is:

```text
Wallet UI / CLI / browser extension / mobile app
                |
        wallet account core
                |
       chain-adapter boundary
       /        |         \
 Sudharma     EVM        Bitcoin
  adapter    adapter      adapter
```

Each adapter will own its chain-specific address derivation, fee model, transaction construction, signing rules, RPC communication, token discovery, and history parsing. Sudharma remains the flagship/native chain and should receive first-class support for SUDH and future Sudharma-native tokens and dApps.

A future recovery redesign should use well-reviewed industry standards for hierarchical deterministic accounts rather than inventing a proprietary universal master-key format. That migration must be separately specified, audited, and backward-compatible with existing encrypted Sudharma wallet files.

## Project-control boundary

Open-source wallet and protocol code does not imply a hidden protocol master key. Official Sudharma network stewardship should instead rely on explicit, auditable controls such as canonical network/genesis parameters, signed official releases, repository/organization governance, treasury custody, upgrade procedures, and later multi-signature or threshold governance where appropriate.
