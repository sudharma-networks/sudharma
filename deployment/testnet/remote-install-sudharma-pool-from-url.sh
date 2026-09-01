#!/usr/bin/env bash
# Installs a prebuilt sudharma-pool binary downloaded from a presigned URL.
set -euo pipefail

pool_url="${SUDHARMA_POOL_BIN_URL:?SUDHARMA_POOL_BIN_URL is required}"
install_path="${SUDHARMA_POOL_INSTALL_PATH:-/usr/local/bin/sudharma-pool}"
config_path="${SUDHARMA_POOL_CONFIG:-/etc/sudharma/pool.json}"
service_name="${SUDHARMA_POOL_SERVICE:-sudharma-pool}"

if [ "$(id -u)" -ne 0 ]; then
  echo "remote-install-sudharma-pool-from-url must run as root" >&2
  exit 2
fi

tmpdir="$(mktemp -d /var/lib/sudharma/pool-install-XXXXXX)"
trap 'rm -rf "$tmpdir"' EXIT

curl --fail --silent --show-error --location "$pool_url" --output "$tmpdir/sudharma-pool"
chmod 0755 "$tmpdir/sudharma-pool"
install -m 0755 "$tmpdir/sudharma-pool" "$install_path"

if [ ! -f "$config_path" ]; then
  echo "Pool config not found at $config_path" >&2
  echo "Install deployment/testnet/pool.example.json before upgrading the binary." >&2
  exit 1
fi

systemctl daemon-reload
if systemctl is-enabled --quiet "$service_name" 2>/dev/null; then
  systemctl restart "$service_name"
  systemctl is-active --quiet "$service_name"
fi

echo '{"sudharma_pool_remote_install":"ok","install_path":"'"$install_path"'"}'
