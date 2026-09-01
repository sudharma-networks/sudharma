<p align="center">
  <img src="assets/sudharma-logo.png" alt="Sudharma Network Logo" width="260">
</p>

<h1 align="center">Sudharma Network</h1>

<p align="center">
  Open-source Proof-of-Work blockchain protocol and development platform.
</p>

<p align="center">
  <strong>Native Coin:</strong> Sudharma (SUDH)
</p>

<p align="center">
  <strong>Website:</strong> <a href="https://feature-website-foundation.d2mqyt0bt8sl9s.amplifyapp.com/">Visit Sudharma Network</a>
</p>

---

## Status

> **Pre-mainnet / Active Development**
>
> Sudharma Network is currently under active development and security hardening. Consensus rules, network parameters, APIs, wallet formats, and other protocol components may change before mainnet.

Do not treat the current software or development network as production-ready.

## Website

The public Sudharma Network website is available at:

**https://feature-website-foundation.d2mqyt0bt8sl9s.amplifyapp.com/**

The website is currently hosted on AWS Amplify and represents the pre-mainnet / active-development network. Features that are still under development are identified as such on the site.

## Overview

Sudharma Network is an open-source blockchain project built around a native Proof-of-Work coin and peer-to-peer network.

The project currently includes blockchain validation, deterministic state management, mining, signed transactions, mempool handling, peer discovery, chain synchronization, reorganization handling, wallet functionality, and automatic peer recovery.

Our long-term goal is to provide an open blockchain development platform on which independent developers can build applications, digital assets, tokens, payment systems, and other decentralized services.

## Native Coin

| Parameter | Value |
| --- | --- |
| Network | Sudharma Network |
| Native Coin | Sudharma |
| Symbol | SUDH |
| Decimal Precision | 8 |
| Maximum Supply (Hard Cap) | 51,000,000,000 SUDH |
| Initial Block Reward | 50 SUDH |
| Target Block Time | 60 seconds |
| Halving Interval | 1,000,000 blocks |
| Premine | 0 |
| Total Transaction Fee | 0.10% |
| Development Portion | 0.01% |
| Miner Portion | 0.09% |

The 51,000,000,000 SUDH value is the consensus hard cap. The current pre-mainnet block-subsidy and halving schedule remains subject to controlled revision before mainnet and cannot mint beyond this cap.

## Current Development Genesis

Current pre-mainnet genesis hash:

```text
ad1535e9780dcbf0bac28fd08722650426889bcdae7fca3bf92bbf257c0c6ec8
```

This is the current **development genesis** and may change before mainnet if consensus-critical parameters are modified.

## Current Features

- Proof-of-Work block validation
- Cumulative-work chain selection
- Deterministic blockchain state
- Signed transactions
- Transaction nonce validation
- Block rewards
- Maximum supply enforcement
- Transaction-fee distribution
- Mempool validation and persistence
- Transaction propagation
- Block propagation
- P2P network handshakes
- Network identity validation
- Peer discovery
- Persisted peer lists
- Automatic peer reconnection
- Chain synchronization
- Blockchain reorganization support
- Wallet creation and storage
- Encrypted wallet support
- Automated blockchain and networking tests

## Repository Structure

```text
blockchain/     Blockchain, state, validation and reorganization
consensus/      Difficulty and reward rules
miner/          Mining pipeline
p2p/            Peer-to-peer networking and synchronization
params/         Network and monetary parameters
pow/            Proof-of-Work functionality
transactions/   Transaction structures and validation
wallet/         Wallet and key-management code

cmd/
  sudharmad/                 Sudharma Network node
  sudharma-wallet/           Sudharma wallet CLI
  sudharma-miner/            GPU miner (solo + Stratum pool worker)
  sudharma-pool/             Reference Stratum pool operator
  sudharma-mining-readiness/ Mining stack readiness report

gpuminer/       GPU miner client library and Stratum worker
pool/           Pool share validation and payout ledgers (PPS/PPLNS/SOLO/FPPS)

assets/         Official project assets
```

## Requirements

- Go 1.26.6 or compatible Go toolchain
- Windows, Linux, or macOS

## Clone

```bash
git clone https://github.com/sudharma-networks/sudharma.git
cd sudharma
```

## Test

Run the complete test suite:

```bash
go test ./... -count=1
```

## Build the Node

```bash
go build ./cmd/sudharmad
```

## Build the Wallet

```bash
go build ./cmd/sudharma-wallet
```

## Run a Local Node

Example:

```bash
go run ./cmd/sudharmad -nodeid node-a -listen 127.0.0.1:18700 -datadir data-node-a
```

Start another node and connect it:

```bash
go run ./cmd/sudharmad -nodeid node-b -listen 127.0.0.1:18701 -peer 127.0.0.1:18700 -datadir data-node-b
```

Runtime blockchain data and private wallet files should never be committed to source control.

## GPU Mining (public-testnet)

Sudharma GPU mining is **GPU-only** (`sudharma-gpupow-v1`). CPU and ASIC mining are rejected.

| Mode | Command |
| --- | --- |
| Solo | `go run ./cmd/sudharma-miner --address YOUR_WALLET` |
| Pool worker | `go run ./cmd/sudharma-miner -stratum stratum+tcp://POOL:3333 -worker rig1` |
| Pool operator | `go run ./cmd/sudharma-pool -config deployment/testnet/pool.example.json` |

Readiness report:

```bash
go run ./cmd/sudharma-mining-readiness
```

Mainnet mining stays closed until launch. See `docs/audits/2026-08-31-pool-mining-architecture.md`.

## Wallet

Display wallet commands:

```bash
go run ./cmd/sudharma-wallet
```

Current wallet operations include:

```text
sudharma-wallet create <wallet-file>
sudharma-wallet address <wallet-file>
sudharma-wallet verify <wallet-file>
```

## Security

Sudharma Network is **not yet mainnet-ready** and has not yet completed an independent security audit.

Current security-development priorities include:

- consensus hardening
- resistance to chain takeover and 51% attacks
- deep-reorganization protection
- exact cumulative-work calculations
- difficulty-adjustment hardening
- timestamp validation
- eclipse-attack resistance
- network partition resistance
- peer scoring and abuse controls
- finality architecture
- adversarial consensus testing

Do not use development wallets to store assets of real-world value.

A formal vulnerability-reporting process will be provided in `SECURITY.md`.

## Open Development Platform

Sudharma Network is intended to remain open for developers, researchers, miners, node operators, and other contributors.

Planned platform capabilities include:

- public node APIs
- RPC infrastructure
- smart-contract execution
- developer SDKs
- token standards
- public testnet
- block explorer
- developer documentation
- third-party decentralized applications

## Official Logo

The official Sudharma Network logo is located at:

```text
assets/sudharma-logo.png
```

## Contributing

Formal contribution guidelines will be provided in `CONTRIBUTING.md`.

Until then, GitHub issues may be used for bugs, technical discussions, and development proposals.

## License

A permanent open-source license has not yet been selected.

The project license will be added separately before the first stable public release.

---

<p align="center">
  <strong>Sudharma Network</strong><br>
  Native Coin: SUDH<br>
  Open Network. Open Development.
</p>