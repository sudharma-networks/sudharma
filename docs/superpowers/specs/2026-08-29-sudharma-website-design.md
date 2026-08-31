# Sudharma Website Design Specification

Date: 2026-08-29
Status: Approved design, pre-implementation
Repository: `sudharma-networks/sudharma`

## 1. Purpose

Build a premium, public-facing Sudharma Network website that presents the project as both a Proof-of-Work blockchain with native coin SUDH and an open development platform for users, miners, developers, researchers, students, and independent builders.

The site must remain accurate to the project's actual readiness. Sudharma is pre-mainnet and under active development. Features that are not complete or verified must be labeled `In Development`, `Experimental`, `Testnet`, or `Planned` rather than presented as production-ready.

The official Sudharma logo from `assets/sudharma-logo.png` is the primary visual identity.

## 2. Positioning and Core Message

Primary positioning:

**Sudharma Network — Open Blockchain. Open Development. Built for Everyone.**

Primary ecosystem paths:

- **Use Sudharma** — wallet, transactions, public testnet, faucet, explorer.
- **Mine Sudharma** — GPU mining, NVIDIA, AMD/cross-vendor support, solo mining, pool mining, Kryptex compatibility, troubleshooting.
- **Build on Sudharma** — source code, nodes, APIs/RPC, wallets, integrations, developer tools, future SDK/token/smart-contract capabilities, contribution.

The website should promote technology, participation, learning, mining, development, and contribution. It must not make investment-return, price-growth, or speculative-profit claims.

## 3. Licensing Message

The repository currently does not define a permanent open-source software license. Until a license is selected and published, the website must avoid promising unrestricted legal reuse.

Allowed language before license finalization:

- open development
- open-source project
- free to participate and contribute
- built for developers and independent projects
- learn from and build with Sudharma subject to the project's published license terms

Once the license is selected, the website may state precise copying, modification, redistribution, and commercial-use rights based on that license.

## 4. Visual Design

The site uses a premium dark technical aesthetic centered on the official Sudharma logo.

Design characteristics:

- deep black/navy background
- restrained luminous accents derived from the official logo rather than arbitrary meme-coin styling
- large, clean typography
- glass-like information cards with subtle depth
- lightweight animated network/node lines
- restrained logo glow
- smooth hover and transition effects
- strong visual hierarchy
- responsive mobile-first layout
- no heavy background video or unnecessary 3D scenes
- motion reduced or disabled for users who prefer reduced motion

The official logo appears consistently in the header, hero, favicon/app icon set, footer, social-preview assets, and relevant product areas.

## 5. Global Navigation

Primary navigation:

`Home | Network | SUDH | Wallet | Mining | Developers | Testnet | Explorer | Faucet | Roadmap | Docs | Community | Support`

Desktop uses a full navigation header. Mobile uses a compact menu with the same destinations.

Every major navigation item and every homepage ecosystem card opens a dedicated routed page. The site must not rely on one oversized single-page document.

A persistent project-readiness badge is shown near the header during the development phase:

`PRE-MAINNET · ACTIVE DEVELOPMENT`

When the public testnet is verified live, the badge may include testnet status, but it must never imply mainnet launch.

## 6. Homepage

### Hero

Headline:

**Open Blockchain. Open Development. Built for Everyone.**

Supporting copy explains that Sudharma is a Proof-of-Work blockchain, native SUDH currency, and open development platform.

Primary actions:

- Explore Sudharma
- Join Public Testnet
- View on GitHub

### Ecosystem Cards

Three large interactive cards:

1. **Use Sudharma**
   - Wallet
   - Transactions
   - Testnet
   - Faucet

2. **Mine Sudharma**
   - GPU Mining
   - NVIDIA
   - AMD
   - Pool Mining

3. **Build on Sudharma**
   - Open Source
   - APIs/RPC
   - Nodes
   - Developer Tools

### Additional Homepage Sections

- More Than a Coin
- Built in the Open
- Current Network / Development Snapshot
- SUDH summary
- Current vs Planned capabilities
- Security and transparency
- Roadmap preview
- Community / contribution call-to-action
- Support / Report Problem call-to-action

Live statistics must only be shown when sourced from verified public APIs. No fabricated counters or demo statistics are allowed in production.

## 7. Network Page

The Network page explains Sudharma visually and technically without turning the page into a pasted whitepaper.

Sections:

- What is Sudharma?
- Proof of Work
- Block validation
- Cumulative-work chain selection
- P2P networking
- Blocks and transactions
- Chain synchronization and reorganization
- Mempool
- Network identity
- Security and current hardening work
- Run a node
- Network parameters

Suggested visual flow:

`Wallet → Transaction → Nodes → Mempool → Miner → Block → Network`

The page should distinguish current capabilities from planned security architecture.

## 8. SUDH Page

The SUDH page presents the native coin using current development parameters from the repository.

Current documented parameters:

- Native Coin: Sudharma
- Symbol: SUDH
- Decimals: 8
- Maximum Supply: 100,000,000 SUDH
- Initial Block Reward: 50 SUDH
- Target Block Time: 60 seconds
- Halving Interval: 1,000,000 blocks
- Premine: 0
- Total Transaction Fee: 0.10%
- Development Portion: 0.01%
- Miner Portion: 0.09%

The page must prominently show:

`PRE-MAINNET PARAMETERS — SUBJECT TO CHANGE BEFORE MAINNET`

Sections include issuance, halving, fees, miner incentives, supply enforcement, and zero-premine explanation.

## 9. Wallet Hub

The Wallet page is designed for nontechnical users first while still exposing technical links.

Sections:

- Android Wallet
- CLI/Desktop wallet where available
- Create wallet
- Receive SUDH
- Send SUDH
- Backup and recovery guidance
- Wallet security
- Testnet connection
- Faucet link
- Verified downloads/releases
- Troubleshooting

Production download buttons must point only to verified releases or controlled deployment infrastructure.

Development wallets must carry clear warnings not to use them for real-world value.

## 10. Mining Hub

Clicking `Mine Sudharma` opens a complete Mining Hub, not a placeholder.

### Mining Hero

- Mine Sudharma
- Proof of Work. Open Participation.
- Current mining-readiness badge

### Mining Sections

- How Sudharma mining works
- Khushi algorithm / current PoW implementation, using repository-verified terminology
- GPU requirements
- NVIDIA mining
- AMD/OpenCL mining
- Other compatible GPU guidance when verified
- Solo mining
- Pool mining
- Kryptex compatibility
- Configuration examples
- Benchmarking
- Performance tuning
- Troubleshooting
- Frequently asked questions
- Mining security and safe downloads
- Testnet mining
- Mining release status

### Detailed Mining Routes

Suggested routes:

- `/mining`
- `/mining/nvidia`
- `/mining/amd`
- `/mining/solo`
- `/mining/pools`
- `/mining/kryptex`
- `/mining/benchmarks`
- `/mining/troubleshooting`

Each route shows only verified hardware/software support. Unverified or unfinished capabilities are labeled clearly.

## 11. Developer Hub

Hero:

**Build on Sudharma — Open infrastructure for the next idea.**

Sections:

- Architecture overview
- Build from source
- Run a node
- RPC/API
- Wallet integration
- Payment integration
- Build developer tools
- Contributing
- GitHub repository
- Developer examples
- Protocol reference
- Security guidance

Planned capabilities are shown in a separate area:

- Developer SDKs
- Token standards
- Smart-contract execution
- Public developer APIs
- third-party dApps

Suggested routes:

- `/developers`
- `/developers/getting-started`
- `/developers/node`
- `/developers/rpc`
- `/developers/wallets`
- `/developers/payments`
- `/developers/contributing`
- `/developers/protocol`

The website must not advertise planned SDK/token/smart-contract capabilities as available until implemented and verified.

## 12. Public Testnet Dashboard

The Testnet page becomes a live dashboard when safe public APIs are available.

Sections/cards:

- Testnet status
- Block height
- Latest block
- Connected/public nodes where appropriate
- Network difficulty where exposed
- Testnet supply where exposed
- Wallet link
- Faucet link
- Mining link
- Explorer link
- Node connection instructions

Permanent disclaimer:

`TESTNET SUDH HAS NO REAL-WORLD MONETARY VALUE.`

The public site must never expose privileged seed-node administration endpoints, credentials, private keys, or internal-only RPC methods.

## 13. Explorer

The Explorer is integrated into the Sudharma visual ecosystem.

Core capabilities:

- Search block / transaction / address
- Latest blocks
- Latest transactions
- Block details
- Transaction details
- Address details
- Network statistics

The Explorer uses a deliberately designed public read-only API layer. It must not connect the browser directly to privileged node administration interfaces.

## 14. Faucet

The Faucet page provides a simple testnet flow:

`Connect/Enter Wallet → Request → Validation → Receive Test SUDH → Show Transaction ID`

Where the challenge system is verified and suitable for public access, the faucet area may also expose the challenge workflow.

The page must clearly state that faucet coins are testnet-only and have no real-world monetary value.

## 15. Roadmap

Roadmap state vocabulary is fixed to:

- `Completed`
- `In Development`
- `Planned`

Roadmap areas may include:

- Blockchain core
- P2P networking
- Public testnet
- Android wallet
- GPU PoW
- Pool/Kryptex compatibility
- Explorer
- Developer APIs
- Security hardening
- Mainnet readiness

Roadmap state must be derived from actual repository/project status rather than marketing goals.

## 16. Documentation

The documentation portal supports:

- Quick Start
- Network
- Node installation
- RPC/API
- Wallet
- Mining
- GPU mining
- Testnet
- Developer guides
- Protocol
- Security
- Contributing
- FAQ

Technical documentation should remain versioned with the repository wherever practical to reduce divergence between code and website documentation.

## 17. Community

The Community area includes only real, official channels.

Sections:

- GitHub
- Report a bug
- Suggest a feature
- Contribute code
- Developer discussions
- Security reporting

Do not display fake user counts, developer counts, transaction counts, community links, or social accounts.

## 18. Support & Problem Reporting Subsystem

The website includes a first-class Support Center accessible from the main navigation and from contextual report buttons across the site.

### Public Support Routes

- `/support`
- `/support/report`
- `/support/track`
- `/support/known-issues`
- `/support/security`
- `/support/faq`

### Report Form

Users can submit:

- affected component
- short title
- detailed written description
- expected behavior
- actual behavior
- steps to reproduce
- environment/device details
- app/node/miner/web version where available
- screenshot
- short video
- log file

Supported component categories include:

- Wallet
- Mining
- Node / Network
- Faucet
- Testnet
- Explorer
- Website
- Developer / API
- Other

### Contextual Reporting

Every major page contains a `Report Problem` action.

If a user reports from a contextual page, the form automatically carries safe metadata such as:

- component
- current route/page
- public application version/build identifier

Example:

`Component: Mining / NVIDIA`

`Page: /mining/nvidia`

No seed phrase, private key, password, AWS credential, privileged token, or sensitive wallet secret is collected automatically.

### Attachment Safety

Before submission, the UI warns users not to upload:

- seed phrases
- private keys
- passwords
- API secrets
- AWS credentials
- other confidential authentication material

Where technically practical, the upload pipeline performs lightweight secret-pattern detection and warns the user before accepting a suspicious attachment. This does not replace server-side security controls.

### GitHub Integration

Ordinary reproducible bugs may create or synchronize with a structured GitHub issue.

Typical issue title:

`[Wallet][Bug] Balance not updating`

Typical labels:

- bug
- wallet
- gpu-mining
- testnet
- faucet
- website
- needs-triage

Users should be given a clear choice where appropriate before potentially personal screenshots/logs are published publicly.

### Private Security Reports

Security/vulnerability reports must not be opened as public GitHub issues by default.

Selecting `Security / Vulnerability` routes the report into a private security workflow. Public disclosure must not occur automatically.

## 19. Mandatory Team Notification Workflow

A report is not treated as successfully processed merely because it was stored.

Required flow:

`User Submission → Secure Report Storage → Issue/Triage Record → Team Notification`

The initial notification architecture should support:

- email alert
- private Sudharma admin/support notification center
- visible unread/new-report badge
- browser/admin popup while an authorized dashboard session is active

Optional future integrations may include Slack, Discord, Telegram, or another official team channel after those channels are formally established.

### Priority

Reports are classified into:

- Critical
- High
- Normal
- Low

Critical and High reports trigger immediate prominent notifications.

### Admin Dashboard

A private support dashboard provides:

- New Reports
- Unread
- Critical
- In Progress
- Resolved

Each record shows report ID, affected component, severity, summary, attachments, GitHub/triage link, status, and history.

Example popup:

`New Sudharma Report — Wallet — High Priority — Balance not updating after transaction`

### Tracking

After a successful submission, the user receives a report identifier such as:

`SUDH-2026-0042`

Public-facing statuses:

`Received → Triaged → Reproduced → Fix in Progress → Fixed → Released`

Tracking must not reveal internal/private security details.

## 20. Optional AI Assistance

AI may assist the reporting workflow but is not the system of record.

User-facing AI assistance may:

- turn an unstructured description into a clear report
- organize expected vs actual behavior
- suggest reproduction steps
- classify the likely affected component
- summarize screenshots/logs where supported

The user reviews the report before submission.

Team-side AI assistance may:

- summarize new reports
- suggest probable duplicates
- propose reproduction checklists
- point engineers toward likely code areas

GitHub/triage records remain authoritative.

## 21. Public/Private Security Boundary

The public website contains no:

- private keys
- wallet seed phrases
- AWS credentials
- GitHub secrets
- privileged RPC credentials
- seed-node admin endpoints
- unrestricted administrative controls

Public live-data features communicate with read-only/public APIs specifically designed for browser consumption.

Administrative support/reporting views require authenticated access and are not bundled as public static data.

## 22. Technical Architecture Direction

The site is one codebase with routed pages and reusable components rather than separate unrelated websites.

Logical architecture:

`Sudharma Web Application`

- Marketing and education pages
- Wallet/download portal
- Mining portal
- Developer documentation
- Testnet dashboard
- Explorer frontend
- Faucet frontend
- Support/reporting frontend

Backend/public services:

- Public read-only network API
- Faucet API
- Explorer API/indexing service as needed
- Support/report submission API
- Attachment storage
- Notification service
- GitHub issue integration
- Private support/admin service

The public static/SSR frontend and privileged backend services remain separated.

## 23. AWS Deployment Direction

Preferred deployment model:

`GitHub → CI/CD → AWS-hosted web frontend/CDN/HTTPS`

AWS Amplify Hosting is the preferred initial public frontend deployment path because it can provide HTTPS, CDN delivery, Git-connected deployment, and custom-domain support with low operational overhead.

Backend services that require credentials, uploads, notifications, or server-side GitHub integration must not be implemented purely in browser JavaScript. They should run behind authenticated/signed server-side AWS components.

The final implementation plan will determine exact AWS services after repository and infrastructure compatibility checks.

## 24. SEO, Accessibility, and Performance

Requirements:

- semantic page titles and descriptions
- social preview metadata using official Sudharma branding
- sitemap and robots policy
- fast first load on Android/mobile networks
- responsive images
- lazy-load noncritical media
- keyboard-accessible navigation
- sufficient contrast
- accessible form labels
- reduced-motion support
- no critical information conveyed through color alone

## 25. Error Handling

User-facing operations must fail clearly and safely.

Examples:

- report upload fails → preserve entered text and show retry
- GitHub synchronization fails → keep internal report and retry server-side; do not lose the user's submission
- notification fails → mark notification delivery as pending/failed and retry; the report still remains stored
- live network API unavailable → show `Data temporarily unavailable`, never fabricated values
- faucet unavailable → show service status and safe retry guidance
- attachment rejected → state size/type/security reason without discarding the rest of the report

## 26. Testing Requirements

The implementation plan must include tests for:

- route/navigation integrity
- responsive/mobile behavior
- accessibility basics
- homepage cards linking to detailed hubs
- detailed Mining routes
- developer routes
- status/readiness labels
- no private credentials bundled into frontend artifacts
- report form validation
- contextual component/page capture
- screenshot/video/log upload constraints
- security-report private routing
- successful internal report creation
- GitHub integration behavior
- notification generation
- notification retry/failure handling
- report tracking states
- API unavailable states
- faucet/explorer boundary behavior
- reduced-motion behavior

End-to-end tests should cover at least one complete normal bug report flow and one private security-report flow.

## 27. Initial Implementation Scope

The first website implementation should prioritize a complete, polished information architecture rather than attempting every future backend capability at once.

Initial implementation should include:

1. shared premium visual system using official Sudharma logo
2. responsive routed public pages
3. Home, Network, SUDH, Wallet, Mining, Developers, Testnet, Explorer, Faucet, Roadmap, Docs, Community, Support shells with accurate content
4. detailed Mining and Developer subroutes
5. readiness labels and pre-mainnet disclaimers
6. Support/Report UI and contextual reporting behavior
7. production-safe backend design hooks for report storage, GitHub integration, and mandatory notifications
8. AWS/GitHub deployment pipeline

Live Explorer, Faucet, network data, report backend, attachments, AI assistance, and private admin dashboard may be delivered in staged milestones where required, but the public UI must never present nonfunctional controls as live without an explicit `Coming Soon`, `In Development`, or disabled state.

## 28. Success Criteria

The website is successful when:

- a first-time visitor understands what Sudharma is within seconds
- users can clearly choose Use, Mine, or Build
- every major navigation tab opens a useful dedicated page
- Mining provides complete structured guidance rather than a shallow card
- developer content clearly separates current functionality from planned capabilities
- the site looks premium and consistently uses the official Sudharma identity
- mobile performance remains strong
- pre-mainnet status is unambiguous
- users can report problems with text and optional media
- reports cannot silently disappear after submission
- the Sudharma team receives direct notification of new reports
- security reports are handled privately
- GitHub remains the authoritative engineering workflow where appropriate
- public frontend code contains no privileged credentials or admin access

## 29. Explicit Non-Goals for the First Release

The first website release will not:

- activate mainnet
- claim mainnet readiness
- promise coin price appreciation or investment returns
- expose privileged seed-node/admin interfaces
- publish security vulnerabilities automatically
- fabricate live network metrics
- claim unsupported GPU/miner compatibility
- claim unrestricted software reuse before the permanent license is selected
- implement unrelated blockchain consensus changes merely to support the website

