#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECKER="$ROOT_DIR/scripts/check-tracked-secrets.sh"

if [[ ! -f "$CHECKER" ]]; then
  echo "FAIL: missing scripts/check-tracked-secrets.sh"
  exit 1
fi

new_repo() {
  local repo
  repo="$(mktemp -d)"
  git -C "$repo" init -q
  git -C "$repo" config user.email "secret-safety-test@example.invalid"
  git -C "$repo" config user.name "Secret Safety Test"
  printf '%s\n' "$repo"
}

cleanup_dirs=()
cleanup() {
  local dir
  for dir in "${cleanup_dirs[@]:-}"; do
    rm -rf "$dir"
  done
}
trap cleanup EXIT

safe_repo="$(new_repo)"
cleanup_dirs+=("$safe_repo")
mkdir -p "$safe_repo/docs" "$safe_repo/wallet"
printf 'safe documentation\n' > "$safe_repo/docs/README.md"
printf 'package wallet\n' > "$safe_repo/wallet/private_key.go"
git -C "$safe_repo" add docs/README.md wallet/private_key.go
bash "$CHECKER" "$safe_repo"

blocked_paths=(
  "admin.pem"
  "keys/node.key"
  "wallets/operator.wallet"
  "funds-treasury.json"
  "wallets/development-wallet.json"
  "wallets/treasury.keystore"
  "recovery/operator.mnemonic"
  "recovery/operator.seed"
  "secrets/operator.secret"
  ".env.production"
  "credentials/operator.p12"
  "credentials/operator.pfx"
  "credentials/operator.jks"
)

for blocked in "${blocked_paths[@]}"; do
  repo="$(new_repo)"
  cleanup_dirs+=("$repo")
  mkdir -p "$repo/$(dirname "$blocked")"
  printf 'dummy test data\n' > "$repo/$blocked"
  git -C "$repo" add -f "$blocked"

  if bash "$CHECKER" "$repo" >/dev/null 2>&1; then
    echo "FAIL: checker allowed tracked secret-like path: $blocked"
    exit 1
  fi
done

echo "PASS: tracked secret safety guard rejects protected file patterns"
