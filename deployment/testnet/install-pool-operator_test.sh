#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
installer="$repo_root/deployment/testnet/install-pool-operator.sh"
unit="$repo_root/deployment/testnet/sudharma-pool.service"
example="$repo_root/deployment/testnet/pool.example.json"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fixtures="$tmp/fixtures"
mkdir -p "$fixtures"
printf '#!/bin/sh\nexit 0\n' > "$fixtures/sudharma-pool"
chmod 0755 "$fixtures/sudharma-pool"

[[ -f "$installer" ]] || { echo "missing installer: $installer" >&2; exit 1; }
[[ -f "$unit" ]] || { echo "missing service unit: $unit" >&2; exit 1; }
[[ -f "$example" ]] || { echo "missing example config: $example" >&2; exit 1; }

run_install() {
  local root="$1"
  shift
  DESTDIR="$root" \
  SUDHARMA_POOL_BIN="$fixtures/sudharma-pool" \
  bash "$installer" "$@"
}

root="$tmp/root-enable"
mkdir -p "$root"
if run_install "$root" --enable >/dev/null 2>&1; then
  echo "--enable with DESTDIR must be refused" >&2
  exit 1
fi

root="$tmp/root-symlink"
mkdir -p "$root/etc/sudharma"
victim="$tmp/outside-config"
printf 'outside-config-must-survive\n' > "$victim"
chmod 0600 "$victim"
ln -s "$victim" "$root/etc/sudharma/pool.json"
if run_install "$root" >/dev/null 2>&1; then
  echo "installer must reject an existing config symlink" >&2
  exit 1
fi

root="$tmp/root-install"
mkdir -p "$root"
run_install "$root" >/dev/null
for required in \
  "$root/usr/local/bin/sudharma-pool" \
  "$root/etc/sudharma/pool.json" \
  "$root/etc/systemd/system/sudharma-pool.service" \
  "$root/var/lib/sudharma-pool"; do
  [[ -e "$required" ]] || { echo "missing installed asset: $required" >&2; exit 1; }
done

grep -Fq 'User=sudharma-pool' "$unit" || { echo "service unit must run as sudharma-pool" >&2; exit 1; }
grep -Fq 'ProtectSystem=strict' "$unit" || { echo "service unit must harden ProtectSystem" >&2; exit 1; }

printf 'PASS: pool operator installer safety checks succeeded\n'
