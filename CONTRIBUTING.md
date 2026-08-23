# Contributing to Sudharma Network

Thank you for your interest in contributing to Sudharma Network.

Sudharma Network is an open-source blockchain project currently in **pre-mainnet active development**.

Because blockchain software contains consensus-critical and security-sensitive components, contributions must be carefully reviewed and tested.

## Ways to Contribute

Contributions may include:

- bug fixes
- documentation
- automated tests
- networking improvements
- wallet improvements
- mining improvements
- performance improvements
- security improvements
- developer tools
- protocol proposals

## Development Setup

Clone the repository:

    git clone https://github.com/sudharma-networks/sudharma.git
    cd sudharma

Run the complete test suite:

    go test ./... -count=1

Format Go code before submitting:

    gofmt -w .

All existing tests should pass before a pull request is submitted.

## Consensus-Critical Changes

Changes affecting any of the following require additional review:

- block validation
- Proof-of-Work
- difficulty adjustment
- cumulative chain work
- fork-choice rules
- chain reorganizations
- genesis rules
- timestamps
- mining rewards
- transaction fees
- maximum supply
- transaction validation
- signatures
- nonces
- replay protection
- blockchain state transitions

A consensus-related proposal should explain the current behavior, proposed behavior, security impact, compatibility impact, possible attack scenarios, and required tests.

Consensus changes may require testnet validation and independent security review before acceptance.

## Security

Never commit:

- private keys
- seed phrases
- passwords
- wallet files
- API secrets
- production credentials

Security vulnerabilities should be reported according to `SECURITY.md`.

Do not publish high-impact exploit details in a normal public GitHub issue.

## Pull Requests

Pull requests should:

- clearly explain what changed
- explain why the change is needed
- include tests when appropriate
- keep unrelated changes separate
- pass the complete test suite
- document compatibility impact when relevant

Submitting code does not automatically make it part of the official Sudharma Network protocol.

All changes to the official repository require maintainer review.

## Backward Compatibility

Sudharma Network is currently pre-mainnet, so breaking changes may still occur.

Changes affecting block formats, transaction formats, wallet formats, P2P protocols, storage formats, network identity, genesis, or consensus rules must clearly document their compatibility impact.

## License

By contributing to this repository, you agree that your contributions may be distributed under the repository's Apache License 2.0.

See `LICENSE` for the complete license terms.

## Conduct

Technical discussions should remain professional and focused on improving the project.

Harassment, threats, spam, abusive behavior, and intentionally disruptive conduct are not acceptable.

---

**Sudharma Network**

Open development with careful consensus and security review.