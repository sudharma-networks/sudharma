#!/usr/bin/env bash
# Patch the seed nginx wallet-proxy allowlist to expose mempool read routes.
set -euo pipefail

CONFIG_PATH="${CONFIG_PATH:-/etc/nginx/sites-enabled/sudharma-wallet-proxy}"
BEGIN_MARKER='# BEGIN SUDHARMA EXPLORER READ-ONLY ALLOWLIST'
END_MARKER='# END SUDHARMA EXPLORER READ-ONLY ALLOWLIST'
SMOKE_BASE="${SMOKE_BASE:-}"
if [ -z "$SMOKE_BASE" ]; then
  seed_ip="$(hostname -I | awk '{print $1}')"
  SMOKE_BASE="http://${seed_ip}:29100"
fi
RPC_SMOKE_BASE="${RPC_SMOKE_BASE:-http://127.0.0.1:28545}"

if [ ! -f "$CONFIG_PATH" ]; then
  echo "nginx wallet proxy config not found: $CONFIG_PATH" >&2
  exit 1
fi

if ! grep -Fq "$BEGIN_MARKER" "$CONFIG_PATH"; then
  echo "explorer allowlist block missing; run rollout-explorer-nginx-allowlist.sh first" >&2
  exit 1
fi

if grep -Fq '/v1/explorer/mempool' "$CONFIG_PATH"; then
  echo "mempool nginx allowlist already present"
else
  backup="$(mktemp "${CONFIG_PATH}.mempool-backup.XXXXXX")"
  work="$(mktemp "${CONFIG_PATH}.mempool-work.XXXXXX")"
  trap 'rm -f "$backup" "$work"' EXIT INT TERM
  cp -p "$CONFIG_PATH" "$backup"

  python3 - "$CONFIG_PATH" "$work" <<'PY'
import pathlib
import sys

source = pathlib.Path(sys.argv[1])
target = pathlib.Path(sys.argv[2])
text = source.read_text()
end_marker = '# END SUDHARMA EXPLORER READ-ONLY ALLOWLIST'
if end_marker not in text:
    raise SystemExit('explorer allowlist end marker missing')

mempool_block = r'''
location = /v1/explorer/mempool {
    limit_except GET { deny all; }
    proxy_http_version 1.1;
    proxy_set_header Host localhost;
    proxy_connect_timeout 2s;
    proxy_read_timeout 5s;
    proxy_pass http://127.0.0.1:28545/v1/explorer/mempool;
}

location = /v1/mempool {
    limit_except GET { deny all; }
    proxy_http_version 1.1;
    proxy_set_header Host localhost;
    proxy_connect_timeout 2s;
    proxy_read_timeout 5s;
    proxy_pass http://127.0.0.1:28545/v1/mempool;
}
'''
target.write_text(text.replace(end_marker, mempool_block + end_marker))
PY

  cp -p "$work" "$CONFIG_PATH"
fi

if ! nginx -t; then
  cp -p "$backup" "$CONFIG_PATH"
  echo 'nginx validation failed; restored previous wallet proxy config' >&2
  exit 1
fi

systemctl reload nginx.service

for attempt in $(seq 1 15); do
  code="$(curl -sS -o /tmp/mempool-smoke.json -w '%{http_code}' "${SMOKE_BASE}/v1/explorer/mempool?limit=1" || true)"
  if [ "$code" = '200' ]; then
    cat /tmp/mempool-smoke.json
    echo "Explorer mempool nginx allowlist ready on ${SMOKE_BASE}"
    exit 0
  fi
  rpc_code="$(curl -sS -o /tmp/mempool-rpc-smoke.json -w '%{http_code}' "${RPC_SMOKE_BASE}/v1/explorer/mempool?limit=1" || true)"
  if [ "$code" = '404' ] && [ "$rpc_code" = '200' ]; then
    echo "nginx allowlist patched; upstream rpcd already serves mempool on ${RPC_SMOKE_BASE}"
    cat /tmp/mempool-rpc-smoke.json
    exit 0
  fi
  sleep 2
done

echo "mempool route smoke check did not succeed (nginx=${code:-unknown}, rpcd=${rpc_code:-unknown})" >&2
exit 1
