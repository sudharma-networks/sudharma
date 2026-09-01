#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: install-pool-operator.sh [--enable]

Installs the Sudharma public-testnet Stratum pool operator service. The service
is installed disabled by default. --enable is an explicit operator action and is
not supported with DESTDIR staging.

Environment overrides:
  DESTDIR         staging root (default: real host root)
  SUDHARMA_POOL_BIN  built sudharma-pool binary
EOF
}

enable=0
case "${1:-}" in
  "") ;;
  --enable) enable=1 ;;
  -h|--help) usage; exit 0 ;;
  *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
esac
[[ $# -le 1 ]] || { usage >&2; exit 2; }

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
destdir="${DESTDIR:-}"
pool_bin="${SUDHARMA_POOL_BIN:-$repo_root/sudharma-pool}"
config_src="$script_dir/pool.example.json"
unit_src="$script_dir/sudharma-pool.service"
config_dest="$destdir/etc/sudharma/pool.json"

if (( enable )) && [[ -n "$destdir" ]]; then
  echo "--enable is refused with DESTDIR staging" >&2
  exit 2
fi

for path in "$pool_bin" "$config_src" "$unit_src"; do
  [[ -f "$path" ]] || { echo "required file not found: $path" >&2; exit 1; }
done
[[ -x "$pool_bin" ]] || { echo "pool binary is not executable: $pool_bin" >&2; exit 1; }

if [[ -L "$config_dest" ]] || { [[ -e "$config_dest" ]] && [[ ! -f "$config_dest" ]]; }; then
  echo "unsafe existing config object: $config_dest" >&2
  exit 2
fi

if [[ -z "$destdir" ]]; then
  if ! id -u sudharma-pool >/dev/null 2>&1; then
    cat >&2 <<'EOF'
Missing dedicated service account sudharma-pool.
Create it before installation, for example:
  sudo useradd --system --home /nonexistent --shell /usr/sbin/nologin sudharma-pool
Then rerun this installer.
EOF
    exit 2
  fi
fi

install -d -m 0755 "$destdir/usr/local/bin"
install -d -m 0755 "$destdir/etc/sudharma"
install -d -m 0755 "$destdir/etc/systemd/system"
install -d -m 0750 "$destdir/var/lib/sudharma-pool"
install -m 0755 "$pool_bin" "$destdir/usr/local/bin/sudharma-pool"
install -m 0644 "$unit_src" "$destdir/etc/systemd/system/sudharma-pool.service"

if [[ ! -e "$config_dest" ]]; then
  install -m 0640 "$config_src" "$config_dest"
else
  chmod 0640 "$config_dest"
  echo "preserved existing config: $config_dest"
fi

if [[ -z "$destdir" ]]; then
  chown sudharma-pool:sudharma-pool /var/lib/sudharma-pool
  chown root:sudharma-pool /etc/sudharma/pool.json
  systemctl daemon-reload
fi

cat <<EOF
Installed Sudharma pool operator assets for dedicated user sudharma-pool.
Service remains disabled unless --enable was explicitly supplied.
Review /etc/sudharma/pool.json and docs/audits/2026-08-31-pool-mining-architecture.md before activation.
EOF

if (( enable )); then
  systemctl enable --now sudharma-pool.service
  systemctl --no-pager --full status sudharma-pool.service
fi
