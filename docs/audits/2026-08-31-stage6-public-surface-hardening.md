# Stage 6 — Public testnet surface hardening

**Recorded:** 2026-08-31  
**Branch:** `cursor/canonical-integration-guard-8441`

## Goal

Make the public website honestly reflect live testnet surfaces and ship a browser faucet
request flow that uses the existing wallet-proxy faucet API.

## In scope

1. `web/lib/faucet-config.ts` + `web/lib/faucet-api.ts` client (info, health, initial grant).
2. `web/components/faucet-panel.tsx` + `/faucet` page UI with address validation and explorer links.
3. Lambda faucet CORS headers and `OPTIONS` preflight handling in
   `deployment/testnet/public-rpc/lambda/index.mjs` (code + unit tests; deploy is operator-gated).
4. Status honesty on home, roadmap, and testnet pages.
5. Public docs for explorer/faucet surfaces (`docs/rpc.md`, `docs/testnet-android.md`).

## Out of scope

- GPU / Khushi staging activation
- Mainnet Tokenomics v1
- Website Amplify static publish (Stage 5 step 5, operator deferred)
- Android APK release (Stage 5 step 7, operator deferred)
- Challenge-round faucet UI on the website (remains Android wallet)

## Operator follow-up

Redeploy the public RPC Lambda (`Testnet Public RPC` workflow) so faucet CORS lands on
the live endpoint. Until then, the website faucet UI ships in git but browser grant
requests may fail CORS against the currently deployed Lambda.
