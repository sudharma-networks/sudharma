#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
installer="$repo_root/deployment/testnet/install-demand-miner.sh"
example_config="$repo_root/deployment/testnet/demand-miner.example.json"
unit="$repo_root/deployment/testnet/sudharma-demand-miner.service"
readme="$repo_root/deployment/testnet/README.md"
workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

require_file() {
  [[ -f "$1" ]] || fail "missing file: $1"
}

require_mode() {
  local path=$1
  local expected=$2
  local actual
  actual=$(stat -c '%a' "$path")
  [[ "$actual" == "$expected" ]] || fail "mode for $path is $actual, want $expected"
}

require_contains() {
  local needle=$1
  local path=$2
  grep -Fqx -- "$needle" "$path" >/dev/null || fail "missing line in $path: $needle"
}

require_not_exists() {
  [[ ! -e "$1" ]] || fail "unexpected path: $1"
}

require_example_config() {
  python3 - "$example_config" <<'PY'
import json
import sys

expected = {
    "environment": "public-testnet",
    "status_url": "http://127.0.0.1:28545",
    "expected_network": "sudharma",
    "expected_coin": "Sudharma",
    "expected_symbol": "SUDH",
    "seed_address": "127.0.0.1:28444",
    "reward_address": "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
    "miner_binary": "/usr/local/libexec/sudharma-demand-miner/sudharmad",
    "data_directory": "/var/lib/sudharma-demand-miner",
    "lock_file": "/run/sudharma-demand-miner/lock",
    "poll_every": "10s",
    "cooldown": "30s",
    "failure_backoff": "30s",
    "child_timeout": "5m",
}
with open(sys.argv[1], encoding="utf-8") as config_file:
    actual = json.load(config_file)
if actual != expected:
    raise SystemExit(f"example config mismatch: {actual!r}")
PY
}

mkdir -p "$workdir/fixtures" "$workdir/bin"
printf '#!/usr/bin/env sh\nexit 0\n' >"$workdir/fixtures/sudharma-demand-miner"
printf '#!/usr/bin/env sh\nexit 0\n' >"$workdir/fixtures/sudharmad"
chmod 0755 "$workdir/fixtures/sudharma-demand-miner" "$workdir/fixtures/sudharmad"
printf '{"fixture":true}\n' >"$workdir/fixtures/demand-miner.json"

cat >"$workdir/bin/systemctl" <<'EOF'
#!/usr/bin/env sh
printf '%s\n' "$*" >>"$SYSTEMCTL_LOG"
EOF
chmod 0755 "$workdir/bin/systemctl"

stage="$workdir/stage"
mkdir -p "$stage/usr/local/bin"
printf 'pre-existing sudharmad sentinel\n' >"$stage/usr/local/bin/sudharmad"
chmod 0755 "$stage/usr/local/bin/sudharmad"
cp "$stage/usr/local/bin/sudharmad" "$workdir/pre-existing-sudharmad"
install_output="$workdir/install.out"
PATH="$workdir/bin:$PATH" SYSTEMCTL_LOG="$workdir/systemctl.log" DESTDIR="$stage" \
  bash "$installer" \
  --miner-binary "$workdir/fixtures/sudharma-demand-miner" \
  --node-binary "$workdir/fixtures/sudharmad" \
  --config "$workdir/fixtures/demand-miner.json" >"$install_output"

require_file "$stage/usr/local/bin/sudharma-demand-miner"
require_file "$stage/usr/local/libexec/sudharma-demand-miner/sudharmad"
require_file "$stage/etc/sudharma/demand-miner.json"
require_file "$stage/etc/systemd/system/sudharma-demand-miner.service"
require_mode "$stage/usr/local/bin/sudharma-demand-miner" 755
require_mode "$stage/usr/local/libexec/sudharma-demand-miner/sudharmad" 755
require_mode "$stage/etc/sudharma/demand-miner.json" 644
require_mode "$stage/etc/systemd/system/sudharma-demand-miner.service" 644
require_mode "$stage/var/lib/sudharma-demand-miner" 750
cmp "$workdir/fixtures/demand-miner.json" "$stage/etc/sudharma/demand-miner.json" >/dev/null
cmp "$workdir/pre-existing-sudharmad" "$stage/usr/local/bin/sudharmad" >/dev/null
require_not_exists "$stage/var/lib/sudharma"
require_contains 'installation complete; service remains disabled' "$install_output"
[[ ! -e "$workdir/systemctl.log" ]] || fail 'installer invoked systemctl while staging'

# A second staged install must be a no-op from the service manager's perspective.
PATH="$workdir/bin:$PATH" SYSTEMCTL_LOG="$workdir/systemctl.log" DESTDIR="$stage" \
  bash "$installer" \
  --miner-binary "$workdir/fixtures/sudharma-demand-miner" \
  --node-binary "$workdir/fixtures/sudharmad" \
  --config "$workdir/fixtures/demand-miner.json" >/dev/null
[[ ! -e "$workdir/systemctl.log" ]] || fail 'idempotent staged installer invoked systemctl'

require_contains 'User=sudharma-miner' "$stage/etc/systemd/system/sudharma-demand-miner.service"
require_contains 'Group=sudharma-miner' "$stage/etc/systemd/system/sudharma-demand-miner.service"
require_contains 'NoNewPrivileges=true' "$stage/etc/systemd/system/sudharma-demand-miner.service"
require_contains 'PrivateTmp=true' "$stage/etc/systemd/system/sudharma-demand-miner.service"
require_contains 'ProtectSystem=strict' "$stage/etc/systemd/system/sudharma-demand-miner.service"
require_contains 'RuntimeDirectory=sudharma-demand-miner' "$stage/etc/systemd/system/sudharma-demand-miner.service"
require_contains 'RuntimeDirectoryMode=0750' "$stage/etc/systemd/system/sudharma-demand-miner.service"
require_contains 'ReadWritePaths=/var/lib/sudharma-demand-miner /run/sudharma-demand-miner' "$stage/etc/systemd/system/sudharma-demand-miner.service"
if grep -q '^ExecStartPre=' "$stage/etc/systemd/system/sudharma-demand-miner.service"; then
  fail 'unit retains an unsafe root ExecStartPre command'
fi
grep -Eq '^ExecStart=.*/flock .*/run/sudharma-demand-miner/lock ' \
  "$stage/etc/systemd/system/sudharma-demand-miner.service" || fail 'unit lacks flock single-instance protection'
require_example_config

grep -Fq 'reward address is public' "$readme" || fail 'README does not disclose reward-address safety'
grep -Fq 'systemctl enable --now sudharma-demand-miner.service' "$readme" || fail 'README lacks explicit enable command'
grep -Fq 'journalctl -u sudharma-demand-miner.service' "$readme" || fail 'README lacks journal command'
grep -Fq 'systemctl status sudharma-demand-miner.service' "$readme" || fail 'README lacks status command'
grep -Fq -- '--rollback' "$readme" || fail 'README lacks rollback command'
grep -Fq 'go build -trimpath -o sudharma-demand-miner ./cmd/sudharma-demand-miner' "$readme" || fail 'README lacks explicit demand-miner build output'
grep -Fq 'go build -trimpath -o sudharmad ./cmd/sudharmad' "$readme" || fail 'README lacks explicit native-miner build output'

# The activation call is deliberately structurally guarded by --enable; normal
# staged installs above prove it never reaches a service manager by default.
awk '
  /if \[\[ "\$enable" == true \]\]; then/ { guarded = 1 }
  guarded && /systemctl enable --now sudharma-demand-miner\.service/ { found = 1 }
  END { exit(found ? 0 : 1) }
' "$installer" || fail 'enable command is not guarded by --enable'
if grep -Eq '/var/lib/sudharma($|[^-])' "$installer"; then
  fail 'rollback or install path references the node data directory'
fi

rollback_output="$workdir/rollback.out"
PATH="$workdir/bin:$PATH" SYSTEMCTL_LOG="$workdir/systemctl.log" DESTDIR="$stage" \
  bash "$installer" --rollback >"$rollback_output"
require_not_exists "$stage/usr/local/bin/sudharma-demand-miner"
require_not_exists "$stage/usr/local/libexec/sudharma-demand-miner/sudharmad"
require_not_exists "$stage/etc/sudharma/demand-miner.json"
require_not_exists "$stage/etc/systemd/system/sudharma-demand-miner.service"
cmp "$workdir/pre-existing-sudharmad" "$stage/usr/local/bin/sudharmad" >/dev/null
[[ -d "$stage/var/lib/sudharma-demand-miner" ]] || fail 'rollback removed demand miner data'
require_not_exists "$stage/var/lib/sudharma"
[[ ! -e "$workdir/systemctl.log" ]] || fail 'staged rollback invoked systemctl'
require_contains 'rollback complete; data directories were preserved' "$rollback_output"

printf 'PASS: demand miner installer safety checks\n'
