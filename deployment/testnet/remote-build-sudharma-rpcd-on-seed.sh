#!/usr/bin/env bash
# Build and install sudharma-rpcd on a seed from the website-foundation branch.
set -euo pipefail

repo_ref="${SUDHARMA_REPO_REF:-feature/website-foundation}"
repo_url="${SUDHARMA_REPO_URL:-https://github.com/sudharma-networks/sudharma.git}"
workdir="${SUDHARMA_RPCD_SRC:-/var/lib/sudharma-rpcd/src}"
service_name="${SUDHARMA_RPCD_SERVICE:-sudharma.service}"
config_path="${SUDHARMA_NODE_CONFIG:-/etc/sudharma/node.json}"
install_path="${SUDHARMA_RPCD_INSTALL_PATH:-/usr/local/bin/sudharma-rpcd}"

if [ "$(id -u)" -ne 0 ]; then
  echo "remote-build-sudharma-rpcd-on-seed must run as root" >&2
  exit 2
fi

for tool in git go curl python3 systemctl install; do
  command -v "$tool" >/dev/null 2>&1 || { echo "required tool missing: $tool" >&2; exit 2; }
done

mkdir -p "$(dirname "$workdir")"
if [ ! -d "$workdir/.git" ]; then
  git clone --depth 1 --branch "$repo_ref" "$repo_url" "$workdir"
else
  git -C "$workdir" fetch origin "$repo_ref"
  git -C "$workdir" checkout "$repo_ref"
  git -C "$workdir" pull --ff-only origin "$repo_ref"
fi

tmpdir="$(mktemp -d /var/lib/sudharma-rpcd/build-XXXXXX)"
export HOME=/var/lib/sudharma-rpcd
export GOPATH=/var/lib/sudharma-rpcd/go
export GOMODCACHE=/var/lib/sudharma-rpcd/go/pkg/mod
export GOCACHE=/var/lib/sudharma-rpcd/go/cache
mkdir -p "$HOME" "$GOMODCACHE" "$GOCACHE"
trap 'rm -rf "$tmpdir"' EXIT

(
  cd "$workdir"
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" -o "$tmpdir/sudharma-rpcd" ./cmd/sudharma-rpcd
)

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
  --arg ref "$repo_ref" \
  '{sudharma_rpcd_install:"ok",service:$service,rpc_port:$port,repo_ref:$ref}'
