#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT="$ROOT_DIR/deployment/testnet/rollout-explorer-nginx-allowlist.sh"

[[ -f "$SCRIPT" ]] || {
  echo "missing rollout script: $SCRIPT" >&2
  exit 1
}
command -v nginx >/dev/null 2>&1 || {
  echo 'nginx is required for the real syntax regression test' >&2
  exit 1
}

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
CONFIG="$TMP_DIR/nginx.conf"

python3 - "$SCRIPT" "$CONFIG" <<'PY'
import pathlib
import sys
import textwrap

script = pathlib.Path(sys.argv[1]).read_text()
out = pathlib.Path(sys.argv[2])
marker = "block = r'''"
start = script.find(marker)
if start < 0:
    raise SystemExit('explorer nginx block marker not found')
start += len(marker)
end = script.find("'''", start)
if end < 0:
    raise SystemExit('explorer nginx block terminator not found')
block = script[start:end].strip('\n')

config = f'''worker_processes 1;
pid {out.parent / "nginx.pid"};
error_log stderr notice;
events {{ worker_connections 32; }}
http {{
    access_log off;
    server {{
        listen 127.0.0.1:18080;
{textwrap.indent(block, "        ")}
        location / {{ return 404; }}
    }}
}}
'''
out.write_text(config)
PY

nginx -t -c "$CONFIG" -p "$TMP_DIR/"
echo 'PASS: real nginx accepts explorer allowlist syntax'
