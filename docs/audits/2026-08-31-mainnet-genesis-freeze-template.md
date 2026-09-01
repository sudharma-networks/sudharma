# Mainnet genesis timestamp freeze — activation PR template

**Status:** Template only. **Do not merge until** independent security audit and launch decision are complete.

This document describes the **only** allowed shape for the genesis-freeze / launch activation PR. It is not authorization to open that PR today.

## Preconditions

- [ ] PR #76 and #77 merged to `main`
- [ ] Independent production security audit recorded (private evidence vault)
- [ ] Written launch decision from project lead
- [ ] Mainnet seed hostnames chosen and DNS ready
- [ ] `go run ./cmd/sudharma-mainnet-readiness` reviewed with all non-human gates green except authorization

## Allowed changes in the activation PR

1. Set `params.MainnetGenesisTimestamp` to the chosen unix timestamp (non-zero, frozen)
2. Optionally set `params.MainnetLaunchAuthorized = true` **only after** human launch decision
3. Optionally set `params.MainnetMiningAuthorized = true` in the same PR or a follow-up PR
4. Update `deployment/mainnet/public-profile.example.json` placeholders → real domains (outside git secrets)
5. Operator runbook verification commands and evidence template references

## Forbidden in the activation PR

- Changing public-testnet genesis, P2P ID, or live reward schedule
- Broad AWS workflow automation (keep `workflow_dispatch` manual gates)
- Faucet / wallet / website changes unrelated to launch
- Setting launch flags without frozen genesis timestamp

## Verification before merge

```bash
go test ./params ./consensus ./blockchain ./p2p ./cmd/sudharmad -count=1
go run ./cmd/sudharma-mainnet-genesis-info | jq .
go run ./cmd/sudharma-mainnet-readiness | jq .
bash ./scripts/mainnet-monetary-rehearsal.sh
bash ./scripts/check-mainnet-readiness-contract_test.sh
```

After `MainnetLaunchAuthorized = true` (staging only):

- `sudharmad -network mainnet` must start against isolated genesis
- Published genesis hash must match `sudharma-mainnet-genesis-info` output
- Issued supply cap must remain **51,000,000 SUDH**

## Evidence to record (private vault)

- Merge commit SHA of activation PR
- `sudharma-mainnet-genesis-info` JSON at freeze time
- Seed deploy workflow run IDs and nginx allowlist confirmation
- Post-deploy smoke: P2P peers, `/v1/status`, monetary cap probe
