#!/usr/bin/env bash
# Install the public, non-secret demand-miner service assets on one host.
set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd "$script_dir/../.." && pwd)
destdir=${DESTDIR:-}
enable=false
rollback=false
miner_binary="$repo_root/sudharma-demand-miner"
node_binary="$repo_root/sudharmad"
config="$script_dir/demand-miner.example.json"

usage() {
  cat <<'EOF'
Usage: install-demand-miner.sh [options]

Install the demand-miner binary, native miner binary, public configuration, and
systemd unit. The service is disabled by default.

Options:
  --miner-binary PATH  Built sudharma-demand-miner binary (default: repository root)
  --node-binary PATH   Built sudharmad binary (default: repository root)
  --config PATH        Demand-miner JSON configuration (default: example config)
  --enable             Enable and start the service after installing it
  --rollback           Remove installed service assets but preserve all data directories
  -h, --help           Show this help

Set DESTDIR to stage files below an alternate root. Staged installations never
create users or invoke systemctl.
EOF
}

fail() {
  printf 'install-demand-miner: %s\n' "$*" >&2
  exit 1
}

target_path() {
  printf '%s%s' "$destdir" "$1"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --miner-binary)
      [[ $# -ge 2 ]] || fail '--miner-binary requires a path'
      miner_binary=$2
      shift 2
      ;;
    --node-binary)
      [[ $# -ge 2 ]] || fail '--node-binary requires a path'
      node_binary=$2
      shift 2
      ;;
    --config)
      [[ $# -ge 2 ]] || fail '--config requires a path'
      config=$2
      shift 2
      ;;
    --enable)
      enable=true
      shift
      ;;
    --rollback)
      rollback=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown option: $1"
      ;;
  esac
done

if [[ -n "$destdir" && "$destdir" != /* ]]; then
  fail 'DESTDIR must be an absolute path'
fi
if [[ -z "$destdir" && $(id -u) -ne 0 ]]; then
  fail 'run as root, or set DESTDIR for a staged installation'
fi

if [[ "$rollback" == true ]]; then
  if [[ -z "$destdir" ]]; then
    systemctl disable --now sudharma-demand-miner.service 2>/dev/null || true
  fi

  rm -f -- \
    "$(target_path /usr/local/bin/sudharma-demand-miner)" \
    "$(target_path /usr/local/libexec/sudharma-demand-miner/sudharmad)" \
    "$(target_path /etc/sudharma/demand-miner.json)" \
    "$(target_path /etc/systemd/system/sudharma-demand-miner.service)"

  if [[ -z "$destdir" ]]; then
    systemctl daemon-reload
  fi
  printf 'rollback complete; data directories were preserved\n'
  exit 0
fi

[[ -f "$miner_binary" ]] || fail "miner binary not found: $miner_binary"
[[ -f "$node_binary" ]] || fail "node binary not found: $node_binary"
[[ -f "$config" ]] || fail "configuration not found: $config"

install -d -m 0755 \
  "$(target_path /usr/local/bin)" \
  "$(target_path /usr/local/libexec/sudharma-demand-miner)" \
  "$(target_path /etc/sudharma)" \
  "$(target_path /etc/systemd/system)"
install -d -m 0750 "$(target_path /var/lib/sudharma-demand-miner)"
install -m 0755 "$miner_binary" "$(target_path /usr/local/bin/sudharma-demand-miner)"
install -m 0755 "$node_binary" "$(target_path /usr/local/libexec/sudharma-demand-miner/sudharmad)"
install -m 0644 "$config" "$(target_path /etc/sudharma/demand-miner.json)"
install -m 0644 "$script_dir/sudharma-demand-miner.service" \
  "$(target_path /etc/systemd/system/sudharma-demand-miner.service)"

if [[ -z "$destdir" ]]; then
  if ! getent group sudharma-miner >/dev/null; then
    groupadd --system sudharma-miner
  fi
  if ! id -u sudharma-miner >/dev/null 2>&1; then
    useradd --system --gid sudharma-miner --home-dir /var/lib/sudharma-demand-miner \
      --shell /usr/sbin/nologin sudharma-miner
  fi
  chown sudharma-miner:sudharma-miner /var/lib/sudharma-demand-miner
  systemctl daemon-reload
fi

if [[ "$enable" == true ]]; then
  if [[ -n "$destdir" ]]; then
    printf 'installation complete; --enable ignored while staging with DESTDIR\n'
    exit 0
  fi
  systemctl enable --now sudharma-demand-miner.service
  printf 'installation complete; service enabled and started\n'
  exit 0
fi

printf 'installation complete; service remains disabled\n'
