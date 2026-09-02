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
| Public testnet maximum supply (legacy hard cap) | 51,000,000,000 SUDH |
| Mainnet maximum supply (final monetary policy) | 51,000,000 SUDH |
| Target Block Time | 60 seconds |
| Mainnet subsidy-bearing blocks | 5,259,600 |
| Mainnet emission epochs | 40 quarterly epochs |
| Mainnet nominal subsidy period | ~10 target years |
| Mainnet subsidy after height 5,259,600 | 0 SUDH |
| Public testnet initial block reward | 50 SUDH |
| Public testnet halving interval | 1,000,000 blocks |
| Premine | 0 |
| Total Transaction Fee | 0.10% |
| Development Portion | 0.01% |
| Miner Portion | 0.09% |

The **final approved mainnet monetary policy is capped at exactly 51,000,000 SUDH** in `params/monetary.go`. It uses a deterministic block-height schedule of 5,259,600 subsidy-bearing blocks split into 40 quarterly epochs, nominally about 10 target years at a 60-second block interval. Subsidy is permanently zero after height 5,259,600. The current public testnet remains on its separate legacy 51,000,000,000 SUDH development policy so the live testnet is not rewritten by this mainnet decision.

Mainnet economics are final, but **mainnet activation is not authorized**. Launch and mining gates remain fail-closed, and the mainnet genesis timestamp remains unset until the required security and readiness gates are completed.

### Mainnet emission roadmap

| Target year | Share of 51M cap | SUDH issued | Cumulative SUDH |
| ---: | ---: | ---: | ---: |
| 1 | 16% | 8,160,000 | 8,160,000 |
| 2 | 14% | 7,140,000 | 15,300,000 |
| 3 | 13% | 6,630,000 | 21,930,000 |
| 4 | 12% | 6,120,000 | 28,050,000 |
| 5 | 11% | 5,610,000 | 33,660,000 |
| 6 | 10% | 5,100,000 | 38,760,000 |
| 7 | 8% | 4,080,000 | 42,840,000 |
| 8 | 7% | 3,570,000 | 46,410,000 |
| 9 | 5% | 2,550,000 | 48,960,000 |
| 10 | 4% | 2,040,000 | 51,000,000 |

Each target year contains 525,960 blocks and four 131,490-block epochs. The duration is block-height based rather than calendar based: if real blocks arrive faster or slower than 60 seconds, the wall-clock completion date moves, but the 51M cap and emission heights do not.

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