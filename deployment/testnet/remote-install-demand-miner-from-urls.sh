#!/usr/bin/env bash
# Installs prebuilt demand-miner binaries downloaded from presigned URLs (no Go on seed).
set -euo pipefail

demand_url="${DEMAND_MINER_BIN_URL:?DEMAND_MINER_BIN_URL is required}"
node_url="${SUDHARMAD_BIN_URL:?SUDHARMAD_BIN_URL is required}"
config_b64="${DEMAND_MINER_CONFIG_B64:-}"
repo_ref="${SUDHARMA_REPO_REF:-feature/faucet-recovery-stage2}"
repo_url="${SUDHARMA_REPO_URL:-https://github.com/sudharma-networks/sudharma.git}"
workdir="${SUDHARMA_DEMAND_MINER_SRC:-/var/lib/sudharma-demand-miner/src}"

if [ "$(id -u)" -ne 0 ]; then
  echo "remote-install-demand-miner-from-urls must run as root" >&2
  exit 2
fi

if ! id -u sudharma-miner >/dev/null 2>&1; then
  useradd --system --home /nonexistent --shell /usr/sbin/nologin sudharma-miner
fi

sync_repo() {
  mkdir -p "$(dirname "$workdir")"
  if [ ! -d "$workdir/.git" ]; then
    git clone --depth 1 --branch "$repo_ref" "$repo_url" "$workdir"
  else
    git -C "$workdir" fetch origin "$repo_ref"
    git -C "$workdir" checkout "$repo_ref"
    git -C "$workdir" pull --ff-only origin "$repo_ref"
  fi
}

tmpdir="$(mktemp -d /var/lib/sudharma-demand-miner/install-XXXXXX)"
trap 'rm -rf "$tmpdir"' EXIT

curl --fail --silent --show-error --location "$demand_url" --output "$tmpdir/sudharma-demand-miner"
curl --fail --silent --show-error --location "$node_url" --output "$tmpdir/sudharmad"
chmod 0755 "$tmpdir/sudharma-demand-miner" "$tmpdir/sudharmad"

install -d -m 0755 /etc/sudharma
if [ -n "$config_b64" ]; then
  echo "$config_b64" | base64 -d > "$tmpdir/demand-miner.json"
  if [ ! -f /etc/sudharma/demand-miner.json ]; then
    install -m 0640 "$tmpdir/demand-miner.json" /etc/sudharma/demand-miner.json
    chown root:sudharma-miner /etc/sudharma/demand-miner.json
  fi
fi

if [ ! -f "$workdir/deployment/testnet/install-demand-miner.sh" ]; then
  sync_repo
fi

if [ ! -f "$workdir/deployment/testnet/install-demand-miner.sh" ]; then
  echo "install-demand-miner.sh not found after repo sync" >&2
  exit 2
fi

DEMAND_MINER_BIN="$tmpdir/sudharma-demand-miner" \
SUDHARMAD_BIN="$tmpdir/sudharmad" \
bash "$workdir/deployment/testnet/install-demand-miner.sh" --enable

if command -v jq >/dev/null 2>&1 && [ -f /etc/sudharma/demand-miner.json ] && [ -f "$workdir/deployment/testnet/demand-miner.seed1-live.example.json" ]; then
  jq -s \
    '.[0] * (.[1] | {poll_every,cooldown,failure_backoff,child_timeout,scheduled_sweep_every,max_blocks_per_sweep})' \
    /etc/sudharma/demand-miner.json "$workdir/deployment/testnet/demand-miner.seed1-live.example.json" \
    > /tmp/demand-miner-schedule.json
  install -m 0640 /tmp/demand-miner-schedule.json /etc/sudharma/demand-miner.json
  chown root:sudharma-miner /etc/sudharma/demand-miner.json
  systemctl restart sudharma-demand-miner.service
fi

systemctl is-active --quiet sudharma-demand-miner.service
curl --fail --silent --max-time 5 http://127.0.0.1:28545/v1/status | tee /tmp/demand-miner-status-after-install.json
printf '{"demand_miner_install":"ok","service":"active"}\n'
