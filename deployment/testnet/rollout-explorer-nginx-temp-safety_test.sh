#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT="$ROOT_DIR/deployment/testnet/rollout-explorer-nginx-allowlist.sh"
[[ -f "$SCRIPT" ]] || { echo "missing rollout script: $SCRIPT" >&2; exit 1; }

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
CONFIG_DIR="$TMP_DIR/sites-enabled"
FAKE_BIN="$TMP_DIR/bin"
mkdir -p "$CONFIG_DIR" "$FAKE_BIN"
CONFIG="$CONFIG_DIR/sudharma-wallet-proxy"
LOG="$TMP_DIR/actions.log"
: > "$LOG"

cat > "$CONFIG" <<'EOF'
server {
    listen 172.31.10.171:29100;
    server_name _;

    location = /v1/status {
        limit_except GET { deny all; }
        proxy_pass http://127.0.0.1:28545/v1/status;
    }

    location / {
        return 404;
    }
}
EOF

cat > "$FAKE_BIN/nginx" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "nginx $*" >> "$FAKE_LOG"
if compgen -G "${CONFIG_PATH}.explorer-*" >/dev/null; then
  echo 'temporary rollout file is visible beside enabled nginx config' >&2
  exit 97
fi
exit 0
EOF
chmod +x "$FAKE_BIN/nginx"

cat > "$FAKE_BIN/systemctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "systemctl $*" >> "$FAKE_LOG"
exit 0
EOF
chmod +x "$FAKE_BIN/systemctl"

cat > "$FAKE_BIN/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
headers=''
output=''
write_out=''
url=''
while (($#)); do
  case "$1" in
    -D|--dump-header) headers="$2"; shift 2 ;;
    -o|--output) output="$2"; shift 2 ;;
    -w|--write-out) write_out="$2"; shift 2 ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$url" in
  */v1/status)
    [[ -z "$output" ]] || printf '{"network":"sudharma"}\n' > "$output"
    [[ -z "$write_out" ]] || printf '200'
    ;;
  */v1/explorer/status)
    [[ -z "$headers" ]] || printf 'Content-Type: application/json; charset=utf-8\r\n\r\n' > "$headers"
    [[ -z "$output" ]] || printf '{"error":"endpoint not found"}\n' > "$output"
    [[ -z "$write_out" ]] || printf '404'
    ;;
  *) exit 22 ;;
esac
EOF
chmod +x "$FAKE_BIN/curl"

if ! PATH="$FAKE_BIN:$PATH" \
  FAKE_LOG="$LOG" \
  CONFIG_PATH="$CONFIG" \
  PRIVATE_IP="172.31.10.171" \
  "$SCRIPT"; then
  echo 'rollout must keep temporary backup/work files outside nginx include directories' >&2
  exit 1
fi

if compgen -G "${CONFIG}.explorer-*" >/dev/null; then
  echo 'temporary rollout files remained beside enabled nginx config' >&2
  exit 1
fi

echo 'PASS: nginx rollout temp files stay outside enabled config directory'
