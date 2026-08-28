#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: install-demand-miner.sh [--enable]

Installs the Sudharma public-testnet demand miner service. The service is
installed disabled by default. --enable is an explicit operator action and is
not supported with DESTDIR staging.

Environment overrides:
  DESTDIR           staging root (default: real host root)
  DEMAND_MINER_BIN  built sudharma-demand-miner binary
  SUDHARMAD_BIN     built sudharmad binary copied to the miner-only libexec path
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
demand_bin="${DEMAND_MINER_BIN:-$repo_root/sudharma-demand-miner}"
node_bin="${SUDHARMAD_BIN:-$repo_root/sudharmad}"
config_src="$script_dir/demand-miner.example.json"
unit_src="$script_dir/sudharma-demand-miner.service"
config_dest="$destdir/etc/sudharma/demand-miner.json"

if (( enable )) && [[ -n "$destdir" ]]; then
  echo "--enable is refused with DESTDIR staging" >&2
  exit 2
fi

for path in "$demand_bin" "$node_bin" "$config_src" "$unit_src"; do
  [[ -f "$path" ]] || { echo "required file not found: $path" >&2; exit 1; }
done
[[ -x "$demand_bin" ]] || { echo "demand miner binary is not executable: $demand_bin" >&2; exit 1; }
[[ -x "$node_bin" ]] || { echo "node binary is not executable: $node_bin" >&2; exit 1; }

# Existing configuration is preserved only when it is a regular file. Reject
# symlinks and other special objects before creating or replacing any assets so
# chmod/chown cannot escape the intended configuration path.
if [[ -L "$config_dest" ]] || { [[ -e "$config_dest" ]] && [[ ! -f "$config_dest" ]]; }; then
  echo "unsafe existing config object: $config_dest" >&2
  exit 2
fi

if [[ -z "$destdir" ]]; then
  if ! id -u sudharma-miner >/dev/null 2>&1; then
    cat >&2 <<'EOF'
Missing dedicated service account sudharma-miner.
Create it before installation, for example:
  sudo useradd --system --home /nonexistent --shell /usr/sbin/nologin sudharma-miner
Then rerun this installer.
EOF
    exit 2
  fi
fi

install -d -m 0755 "$destdir/usr/local/bin"
install -d -m 0755 "$destdir/usr/local/libexec/sudharma-demand-miner"
install -d -m 0755 "$destdir/etc/sudharma"
install -d -m 0755 "$destdir/etc/systemd/system"
install -d -m 0750 "$destdir/var/lib/sudharma-demand-miner"
install -m 0755 "$demand_bin" "$destdir/usr/local/bin/sudharma-demand-miner"
install -m 0755 "$node_bin" "$destdir/usr/local/libexec/sudharma-demand-miner/sudharmad"
install -m 0644 "$unit_src" "$destdir/etc/systemd/system/sudharma-demand-miner.service"

if [[ ! -e "$config_dest" ]]; then
  install -m 0640 "$config_src" "$config_dest"
else
  chmod 0640 "$config_dest"
  echo "preserved existing config: $config_dest"
fi

if [[ -z "$destdir" ]]; then
  chown sudharma-miner:sudharma-miner /var/lib/sudharma-demand-miner
  chown root:sudharma-miner /etc/sudharma/demand-miner.json
  systemctl daemon-reload
fi

cat <<EOF
Installed Sudharma demand miner assets for dedicated user sudharma-miner.
Service remains disabled unless --enable was explicitly supplied.
Review /etc/sudharma/demand-miner.json and the deployment runbook before activation.
EOF

if (( enable )); then
  systemctl enable --now sudharma-demand-miner.service
  systemctl --no-pager --full status sudharma-demand-miner.service
fi
