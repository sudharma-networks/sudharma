#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

require_file() {
  [ -f "$1" ] || fail "$1 is missing"
}

for file in \
  deployment/mainnet/README.md \
  deployment/mainnet/OPERATOR-CHECKLIST.md \
  deployment/mainnet/deployment-evidence.template.json \
  deployment/mainnet/gpu-miner.example.json \
  deployment/mainnet/gpu-miner-pool.example.json \
  deployment/mainnet/pool.example.json \
  deployment/mainnet/gpu-miner.seed1-live.example.json \
  deployment/mainnet/gpu-miner.seed2-live.example.json \
  deployment/mainnet/seed1.node.example.json \
  deployment/mainnet/seed2.node.example.json \
  deployment/mainnet/public-profile.example.json \
  deployment/mainnet/nginx-rpc.example.conf \
  deployment/mainnet/docker-compose.example.yml \
  deployment/mainnet/sudharma-mainnet.service \
  deployment/testnet/install-pool-operator.sh \
  deployment/testnet/pool-operator-runbook.md \
  deployment/testnet/remote-install-sudharma-pool-from-url.sh \
  docs/audits/2026-08-31-mainnet-merge-review-checklist.md \
  docs/audits/2026-08-31-mainnet-genesis-freeze-template.md \
  docs/audits/2026-08-31-mainnet-launch-operator-runbook.md \
  docs/audits/2026-08-31-mainnet-gpu-mining-architecture.md \
  docs/audits/2026-08-31-pool-mining-architecture.md; do
  require_file "$file"
done

grep -Fq 'MainnetMiningAuthorized = false' params/mining.go \
  || fail 'mainnet GPU mining must stay unauthorized in params/mining.go'

grep -Fq 'MainnetSeed1RPC' params/mining.go \
  || fail 'mainnet seed RPC placeholders must be encoded in params/mining.go'

grep -Fq 'REPLACE_WITH_REAL_DOMAIN' deployment/mainnet/public-profile.example.json \
  || fail 'mainnet public profile must keep unresolved placeholders until launch'

grep -Fq 'buildPOWCompatWork' rpc/mining_compat.go \
  || fail 'PoW compatibility aliases must be encoded in rpc/mining_compat.go'

grep -Fq 'NewChainFor' blockchain/chain.go \
  || fail 'network-aware chain constructor must exist in blockchain/chain.go'

grep -Fq 'SetLocalNetworkID' p2p/network.go \
  || fail 'network-aware P2P handshake selection must exist in p2p/network.go'

grep -Fq 'NewStateFor' blockchain/state.go \
  || fail 'policy-bound state constructor must exist in blockchain/state.go'

grep -Fq 'EnsureMonetaryPolicy' blockchain/state.go \
  || fail 'ProcessBlockFor must validate monetary policy against state'

grep -Fq 'ProcessBlockFor' cmd/sudharmad/main.go \
  || fail 'sudharmad must replay blocks with ProcessBlockFor'

require_file scripts/probe-testnet-mining-rpc.sh

go test ./blockchain -run 'TestNewChainFor|TestValidateChainGenesis|TestMintSupplyForMainnetEnforcesMainnetCap' -count=1 >/dev/null \
  || fail 'network-aware chain and monetary policy tests must pass'

go test ./p2p -run 'TestSetLocalNetworkID' -count=1 >/dev/null \
  || fail 'network-aware P2P tests must pass'

go test ./gpuminer -run 'TestLoadMainnetFileConfigMatchesOperatorShape|TestLoadFileConfigMatchesDemandMinerShape' -count=1 >/dev/null \
  || fail 'mainnet and testnet gpu-miner deployment configs must parse'

printf 'PASS: mainnet go-live operator toolkit scaffold is present\n'
