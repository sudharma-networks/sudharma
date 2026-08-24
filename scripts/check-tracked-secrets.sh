#!/usr/bin/env bash
set -euo pipefail

repo_dir="${1:-.}"

if ! git -C "$repo_dir" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "ERROR: not a Git work tree: $repo_dir" >&2
  exit 2
fi

violations=()

while IFS= read -r -d '' tracked_path; do
  lower_path="${tracked_path,,}"
  base_name="${lower_path##*/}"

  case "$lower_path" in
    *.pem|*.key|*.wallet|*-treasury.json|*-treasury.wallet|*-development-wallet.json|*-dev-wallet.json|*.keystore|*.mnemonic|*.seed|*.seedphrase|*.recovery|*.p12|*.pfx|*.jks|*.secret)
      violations+=("$tracked_path")
      continue
      ;;
    secrets/*|*/secrets/*|wallets/*|*/wallets/*)
      violations+=("$tracked_path")
      continue
      ;;
  esac

  case "$base_name" in
    .env|.env.*)
      violations+=("$tracked_path")
      ;;
  esac
done < <(git -C "$repo_dir" ls-files -z)

if (( ${#violations[@]} > 0 )); then
  echo "ERROR: tracked secret-like files are not allowed:" >&2
  printf '  - %s\n' "${violations[@]}" >&2
  echo "Remove them from Git tracking and rotate any exposed credential before continuing." >&2
  exit 1
fi

echo "PASS: no prohibited secret-like files are tracked"
