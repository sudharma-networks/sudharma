#!/usr/bin/env bash
# Installs a prebuilt sudharma-rpcd binary downloaded from a presigned URL.
set -euo pipefail

rpcd_url="${SUDHARMA_RPCD_BIN_URL:?SUDHARMA_RPCD_BIN_URL is required}"
service_name="${SUDHARMA_RPCD_SERVICE:-sudharma.service}"
config_path="${SUDHARMA_NODE_CONFIG:-/etc/sudharma/node.json}"
install_path="${SUDHARMA_RPCD_INSTALL_PATH:-/usr/local/bin/sudharma-rpcd}"

if [ "$(id -u)" -ne 0 ]; then
  echo "remote-install-sudharma-rpcd-from-url must run as root" >&2
  exit 2
fi

tmpdir="$(mktemp -d /var/lib/sudharma/rpcd-install-XXXXXX)"
trap 'rm -rf "$tmpdir"' EXIT

curl --fail --silent --show-error --location "$rpcd_url" --output "$tmpdir/sudharma-rpcd"
chmod 0755 "$tmpdir/sudharma-rpcd"
install -m 0755 "$tmpdir/sudharma-rpcd" "$install_path"

if [ ! -f "$config_path" ]; then
  echo "Node config not found at $config_path" >&2
  exit 1
fi

rpc_port="$(python3 - <<'PY' "$config_path"
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    cfg = json.load(fh)
addr = str(cfg.get("rpc_address", "127.0.0.1:28545"))
print(addr.rsplit(":", 1)[-1])
PY
)"

systemctl daemon-reload
systemctl restart "$service_name"
systemctl is-active --quiet "$service_name"

for attempt in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:${rpc_port}/ready" >/dev/null 2>&1; then
    break
  fi
  if [ "$attempt" -eq 30 ]; then
    echo "sudharma-rpcd did not become ready on port ${rpc_port}" >&2
    systemctl status "$service_name" --no-pager || true
    exit 1
  fi
  sleep 2
done

curl -fsS "http://127.0.0.1:${rpc_port}/v1/explorer/mempool?limit=1" | tee /tmp/sudharma-rpcd-mempool-smoke.json
curl -fsS "http://127.0.0.1:${rpc_port}/v1/mempool?limit=1" | tee /tmp/sudharma-rpcd-raw-mempool-smoke.json

jq -nc \
  --arg service "$service_name" \
  --arg port "$rpc_port" \
  --arg install_path "$install_path" \
  '{sudharma_rpcd_install:"ok",service:$service,rpc_port:$port,install_path:$install_path}'
