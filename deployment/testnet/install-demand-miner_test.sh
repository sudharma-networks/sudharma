#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
installer="$repo_root/deployment/testnet/install-demand-miner.sh"
unit="$repo_root/deployment/testnet/sudharma-demand-miner.service"
example="$repo_root/deployment/testnet/demand-miner.example.json"
readme="$repo_root/deployment/testnet/README.md"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
root="$tmp/root"
fixtures="$tmp/fixtures"
mkdir -p "$fixtures" "$root/usr/local/bin"
printf '#!/bin/sh\nexit 0\n' > "$fixtures/sudharma-demand-miner"
printf '#!/bin/sh\nexit 0\n' > "$fixtures/sudharmad"
printf 'existing-node-binary-must-survive\n' > "$root/usr/local/bin/sudharmad"
shared_node_before="$(sha256sum "$root/usr/local/bin/sudharmad" | awk '{print $1}')"
chmod 0755 "$fixtures/sudharma-demand-miner" "$fixtures/sudharmad"

[[ -f "$installer" ]] || { echo "missing installer: $installer" >&2; exit 1; }
[[ -f "$unit" ]] || { echo "missing service unit: $unit" >&2; exit 1; }
[[ -f "$example" ]] || { echo "missing example config: $example" >&2; exit 1; }

run_install() {
  DESTDIR="$root" \
  DEMAND_MINER_BIN="$fixtures/sudharma-demand-miner" \
  SUDHARMAD_BIN="$fixtures/sudharmad" \
  bash "$installer" "$@"
}

if run_install --enable >/dev/null 2>&1; then
  echo "--enable with DESTDIR must be refused" >&2
  exit 1
fi
for forbidden in \
  "$root/usr/local/bin/sudharma-demand-miner" \
  "$root/usr/local/libexec/sudharma-demand-miner/sudharmad" \
  "$root/etc/sudharma/demand-miner.json" \
  "$root/etc/systemd/system/sudharma-demand-miner.service" \
  "$root/var/lib/sudharma-demand-miner"; do
  if [[ -e "$forbidden" ]]; then
    echo "refused DESTDIR --enable must not mutate staging root: $forbidden" >&2
    exit 1
  fi
done

output="$(run_install)"
run_install >/dev/null

assert_mode() {
  local want="$1" path="$2" got
  got="$(stat -c '%a' "$path")"
  [[ "$got" == "$want" ]] || { echo "mode $path: want $want got $got" >&2; exit 1; }
}

[[ -f "$root/usr/local/bin/sudharma-demand-miner" ]]
[[ -f "$root/usr/local/libexec/sudharma-demand-miner/sudharmad" ]]
[[ -f "$root/etc/sudharma/demand-miner.json" ]]
[[ -f "$root/etc/systemd/system/sudharma-demand-miner.service" ]]
[[ -d "$root/var/lib/sudharma-demand-miner" ]]
assert_mode 755 "$root/usr/local/bin/sudharma-demand-miner"
assert_mode 755 "$root/usr/local/libexec/sudharma-demand-miner/sudharmad"
assert_mode 640 "$root/etc/sudharma/demand-miner.json"
assert_mode 644 "$root/etc/systemd/system/sudharma-demand-miner.service"
assert_mode 750 "$root/var/lib/sudharma-demand-miner"

shared_node_after="$(sha256sum "$root/usr/local/bin/sudharmad" | awk '{print $1}')"
[[ "$shared_node_after" == "$shared_node_before" ]] || {
  echo "installer must not replace shared /usr/local/bin/sudharmad" >&2
  exit 1
}

grep -Fq '"status_url": "http://127.0.0.1:28545"' "$root/etc/sudharma/demand-miner.json"
grep -Fq '"miner_binary": "/usr/local/libexec/sudharma-demand-miner/sudharmad"' "$root/etc/sudharma/demand-miner.json"
grep -Fq '"data_directory": "/var/lib/sudharma-demand-miner"' "$root/etc/sudharma/demand-miner.json"
grep -Fq '"lock_file": "/run/sudharma-demand-miner/lock"' "$root/etc/sudharma/demand-miner.json"
grep -Fq 'User=sudharma-miner' "$root/etc/systemd/system/sudharma-demand-miner.service"
grep -Fq 'NoNewPrivileges=true' "$root/etc/systemd/system/sudharma-demand-miner.service"
grep -Fq 'PrivateTmp=true' "$root/etc/systemd/system/sudharma-demand-miner.service"
grep -Fq 'ProtectSystem=strict' "$root/etc/systemd/system/sudharma-demand-miner.service"
grep -Fq 'RuntimeDirectory=sudharma-demand-miner' "$root/etc/systemd/system/sudharma-demand-miner.service"
grep -Fq 'RuntimeDirectoryMode=0750' "$root/etc/systemd/system/sudharma-demand-miner.service"
grep -Fq 'ReadWritePaths=/var/lib/sudharma-demand-miner' "$root/etc/systemd/system/sudharma-demand-miner.service"
grep -Fq 'ReadWritePaths=/run/sudharma-demand-miner' "$root/etc/systemd/system/sudharma-demand-miner.service"
grep -Fq 'ExecStart=/usr/local/bin/sudharma-demand-miner -config /etc/sudharma/demand-miner.json' "$root/etc/systemd/system/sudharma-demand-miner.service"
if grep -Fq 'ExecStartPre=' "$root/etc/systemd/system/sudharma-demand-miner.service"; then
  echo "service must not replace the lock inode during startup" >&2
  exit 1
fi

grep -Fq 'sudharma-miner' <<<"$output"
if grep -Fq 'systemctl enable --now' <<<"$output"; then
  echo "default install must not enable service" >&2
  exit 1
fi
if grep -Eq 'systemctl[[:space:]]+enable[[:space:]]+--now' "$installer" && ! grep -Fq -- '--enable' "$installer"; then
  echo "enable --now must be gated by --enable" >&2
  exit 1
fi

rollback="$(awk '/^### Rollback/{flag=1; next} flag && /^### /{exit} flag{print}' "$readme")"
[[ -n "$rollback" ]] || { echo "missing Rollback section" >&2; exit 1; }
if grep -Fq '/var/lib/sudharma ' <<<"$rollback" || grep -Fq '/var/lib/sudharma/' <<<"$rollback"; then
  echo "rollback must never reference node data /var/lib/sudharma" >&2
  exit 1
fi

echo "demand miner installer safety checks passed"
