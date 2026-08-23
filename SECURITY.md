# Security Policy

## Project Status

Sudharma Network is currently in **pre-mainnet active development**.

The protocol, consensus rules, networking, wallet formats, APIs, and security architecture are still being developed and hardened.

The current software should not be treated as production-ready. Do not use development builds or development wallets to store assets of real-world value.

## Reporting a Security Vulnerability

If you discover a security vulnerability in Sudharma Network, please **do not disclose it publicly** until it has been reviewed and addressed.

Do not open a normal public GitHub issue for a high-impact vulnerability.

Examples include:

- consensus bypasses
- transaction or signature forgery
- private-key or wallet vulnerabilities
- chain-reorganization vulnerabilities
- Proof-of-Work validation flaws
- network takeover techniques
- remote code execution
- serious denial-of-service vulnerabilities
- supply or reward calculation vulnerabilities

## How to Report

Please report security vulnerabilities privately through GitHub.

Use GitHub's **private vulnerability reporting** feature for the Sudharma repository when available.

Repository:

https://github.com/sudharma-networks/sudharma

If private vulnerability reporting is unavailable, contact the repository maintainers privately rather than publishing exploit details.

## What to Include

Please include as much of the following as possible:

- affected component
- affected version or commit
- vulnerability description
- reproduction steps
- expected behavior
- actual behavior
- potential impact
- proof-of-concept information when appropriate
- suggested mitigation if known

## Security Areas of Interest

Sudharma Network welcomes responsible security research involving:

- consensus correctness
- Proof-of-Work validation
- difficulty adjustment
- cumulative chain-work calculations
- fork-choice rules
- chain reorganizations
- deep-reorganization attacks
- 51% attack resistance
- timestamp manipulation
- transaction validation
- signature verification
- nonce and replay protection
- supply enforcement
- mining rewards and fee distribution
- mempool security
- P2P protocol security
- eclipse and Sybil attacks
- malicious peers
- chain synchronization
- wallet encryption and private-key handling
- denial-of-service resistance

## Current Security Priorities

Before mainnet, development priorities include:

- exact target-derived chain-work calculation
- deep-reorganization protection
- difficulty and timestamp hardening
- 51% attack resistance
- eclipse-attack resistance
- peer scoring and abuse controls
- network partition recovery
- adversarial consensus testing
- wallet security
- fuzz testing
- independent security review

## Coordinated Disclosure

Please allow reasonable time for the maintainers to investigate, reproduce, fix, test, and deploy a solution before public disclosure.

After users and the network are no longer at immediate risk, coordinated disclosure may be appropriate.

## Security Research

Security research should preferably be performed on local networks, isolated environments, or designated testnets.

Researchers should avoid:

- accessing data that does not belong to them
- stealing or moving funds
- disrupting public infrastructure
- intentionally degrading network availability
- testing third-party systems without permission

## Bug Bounty

Sudharma Network does not currently operate a formal paid bug bounty program.

Submitting a vulnerability does not guarantee financial compensation.

## Contact

Official repository:

https://github.com/sudharma-networks/sudharma

Official organization:

https://github.com/sudharma-networks

---

**Sudharma Network**

Security-first development before mainnet.