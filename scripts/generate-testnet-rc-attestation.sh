#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo 'generate-testnet-rc-attestation: not inside a git repository' >&2
  exit 1
fi

commit="$(git rev-parse HEAD)"
branch="$(git rev-parse --abbrev-ref HEAD)"
generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

run_check() {
  local name="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    printf '%s\n' "$name:pass"
  else
    printf '%s\n' "$name:fail"
  fi
}

checks="$(
  run_check canonical_faucet_recovery bash ./scripts/check-canonical-faucet-recovery-contract_test.sh
  run_check explorer_api bash ./scripts/check-explorer-api-contract_test.sh
  run_check faucet_deploy bash ./scripts/check-faucet-deploy-contract_test.sh
  run_check demand_miner_auto_deploy bash ./scripts/check-demand-miner-auto-deploy-contract_test.sh
  run_check workflow_safety node --test ./scripts/live-workflow-trigger-safety.test.mjs
)"

lambda_dir='deployment/testnet/public-rpc/lambda'
lambda_sha=''
if [ -d "$lambda_dir" ]; then
  lambda_sha="$(find "$lambda_dir" -type f \( -name '*.mjs' -o -name 'package.json' \) -print0 \
    | sort -z \
    | xargs -0 sha256sum \
    | sha256sum \
    | awk '{print $1}')"
fi

web_sha=''
if [ -f web/package-lock.json ]; then
  web_sha="$(sha256sum web/package-lock.json | awk '{print $1}')"
fi

go_build_ok='false'
if go build -trimpath ./cmd/sudharmad ./cmd/sudharma-rpcd ./cmd/sudharma-demand-miner >/dev/null 2>&1; then
  go_build_ok='true'
fi

python3 - "$commit" "$branch" "$generated_at" "$checks" "$lambda_sha" "$web_sha" "$go_build_ok" <<'PY'
import json, sys

commit, branch, generated_at, checks, lambda_sha, web_sha, go_build_ok = sys.argv[1:8]
check_rows = []
for line in checks.splitlines():
    if not line.strip():
        continue
    name, status = line.split(':', 1)
    check_rows.append({"name": name, "status": status})

payload = {
    "kind": "sudharma-testnet-release-candidate-attestation",
    "generated_at": generated_at,
    "candidate": {
        "git_commit": commit,
        "git_branch": branch,
    },
    "contract_checks": check_rows,
    "artifact_fingerprints": {
        "lambda_source_tree_sha256": lambda_sha or None,
        "web_package_lock_sha256": web_sha or None,
        "go_build_ok": go_build_ok == "true",
    },
    "deployment_evidence_required_before_go_live": [
        "exact running commit or artifact digest for each seed service",
        "exact Lambda code/configuration revision and signer-independent health data",
        "exact demand-miner binary revision for both seeds",
        "exact website build revision",
        "APK tag, source revision, and checksum for every published wallet",
        "reviewed diff from deployed revisions to this candidate commit",
    ],
    "operator_authorization": "This attestation does not authorize AWS changes, service restarts, faucet payouts, mining actions, release publication, or consensus activation.",
}

print(json.dumps(payload, indent=2))
PY
