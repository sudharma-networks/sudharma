#!/usr/bin/env bash
# Enables /v1/mining/work and /v1/mining/submit on the seed nginx listener
# that fronts sudharma-rpcd (typically port 29100).
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "remote-enable-seed-mining-nginx must run as root" >&2
  exit 2
fi

discover_config_files() {
  local -a files=()
  local path
  while IFS= read -r path; do
    [ -n "$path" ] && files+=("$path")
  done < <(
    grep -RIl --include='*.conf' --include='*.inc' -E '/v1/status|/v1/explorer/' /etc/nginx 2>/dev/null | sort -u
  )
  if [ "${#files[@]}" -eq 0 ] && command -v nginx >/dev/null 2>&1; then
    while IFS= read -r path; do
      [ -n "$path" ] && files+=("$path")
    done < <(nginx -T 2>/dev/null | awk -F: '/^# configuration file / {print $2}' | sed 's/:$//' | sort -u)
  fi
  if [ "${#files[@]}" -eq 0 ]; then
    while IFS= read -r path; do
      [ -n "$path" ] && files+=("$path")
    done < <(
      grep -RIl --include='*.conf' --include='*.inc' '29100' /etc/nginx /etc/sudharma /usr/local/etc/nginx 2>/dev/null | sort -u
    )
  fi
  printf '%s\n' "${files[@]}"
}

mapfile -t config_files < <(discover_config_files)

if [ "${#config_files[@]}" -eq 0 ]; then
  echo "Could not locate sudharma public RPC nginx config" >&2
  if command -v nginx >/dev/null 2>&1; then
    nginx -T 2>/dev/null | grep -E 'listen|/v1/status|/v1/explorer|29100' | head -80 >&2 || true
  fi
  exit 1
fi

python3 - <<'PY' "${config_files[@]}"
import json
import pathlib
import re
import sys

configs = [pathlib.Path(p) for p in sys.argv[1:] if p]
snippet = """
    location = /v1/mining/work {
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 3s;
        proxy_read_timeout 30s;
        proxy_pass UPSTREAM;
    }

    location = /v1/mining/submit {
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        client_max_body_size 1m;
        proxy_connect_timeout 3s;
        proxy_read_timeout 30s;
        proxy_pass UPSTREAM;
    }
"""

def infer_upstream(text: str) -> str | None:
    patterns = (
        r"location\s+=\s+/v1/status\s*\{[^}]*?proxy_pass\s+([^;]+);",
        r"location\s+/v1/explorer/\s*\{[^}]*?proxy_pass\s+([^;]+);",
        r"location\s+/v1/status\s*\{[^}]*?proxy_pass\s+([^;]+);",
        r"location\s+~\s+\^/v1/\([^\)]*\)[^{]*\{[^}]*?proxy_pass\s+([^;]+);",
        r"proxy_pass\s+(http://127\.0\.0\.1:[0-9]+)",
    )
    for pattern in patterns:
        match = re.search(pattern, text, re.S)
        if match:
            return match.group(1).strip()
    return None

def patch_regex_allowlist(text: str) -> tuple[str, bool]:
    pattern = re.compile(
        r"(location\s+~\s+\^/v1/\()([^)]+)(\)[^{]*\{[^}]*?proxy_pass\s+[^;]+;[^}]*\})",
        re.S,
    )
    match = pattern.search(text)
    if not match or "mining" in match.group(2):
        return text, False
    updated_group = match.group(2).rstrip("|") + "|mining/work|mining/submit|"
    return text[:match.start(2)] + updated_group + text[match.end(2):], True

def patch_exact_locations(text: str, upstream: str) -> tuple[str, bool]:
    if 'location = /v1/mining/work' in text and 'location = /v1/mining/submit' in text:
        return text, False
    marker = "    # sudharma gpu mining routes"
    if marker in text:
        return text, False
    anchor = text.find("/v1/status")
    if anchor == -1:
        anchor = text.find("/v1/explorer/")
    if anchor == -1:
        return text, False
    server_start = text.rfind("server", 0, anchor)
    if server_start == -1:
        return text, False
    brace = text.find("{", server_start)
    if brace == -1:
        return text, False
    depth = 0
    end = None
    for i in range(brace, len(text)):
        ch = text[i]
        if ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                end = i
                break
    if end is None:
        return text, False
    insert = snippet.replace("UPSTREAM", upstream)
    return text[:end] + marker + insert + text[end:], True

changed = []
for path in configs:
    if not path.is_file():
        continue
    text = path.read_text(encoding="utf-8")
    updated, ok = patch_regex_allowlist(text)
    if ok:
        path.write_text(updated, encoding="utf-8")
        changed.append(str(path))
        continue
    upstream = infer_upstream(text)
    if not upstream:
        continue
    updated, ok = patch_exact_locations(text, upstream)
    if ok:
        path.write_text(updated, encoding="utf-8")
        changed.append(str(path))

if not changed:
    print(json.dumps({"seed_mining_nginx": "already_enabled"}))
else:
    print(json.dumps({"seed_mining_nginx": "updated", "files": changed}))
PY

nginx -t
systemctl reload nginx

smoke_candidates=()
if [ -n "${SEED_PRIVATE_IP:-}" ]; then
  smoke_candidates+=("http://${SEED_PRIVATE_IP}:29100")
fi
if [ -n "${SEED_RPC_SMOKE_URL:-}" ]; then
  smoke_candidates+=("${SEED_RPC_SMOKE_URL%/}")
fi
smoke_candidates+=("http://127.0.0.1:29100")

node_config="${SUDHARMA_NODE_CONFIG:-/etc/sudharma/node.json}"
if [ -f "$node_config" ]; then
  rpc_port="$(python3 - <<'PY' "$node_config"
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    cfg = json.load(fh)
print(str(cfg.get("rpc_address", "127.0.0.1:28545")).rsplit(":", 1)[-1])
PY
)"
  smoke_candidates+=("http://127.0.0.1:${rpc_port}")
fi

for smoke_base in "${smoke_candidates[@]}"; do
  for attempt in $(seq 1 8); do
    if curl -fsS -X POST "${smoke_base}/v1/mining/work" \
      -H 'content-type: application/json' \
      --data '{"address":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}' \
      | tee /tmp/seed-mining-work-smoke.json \
      | jq -e '.algorithm and .block' >/dev/null 2>&1; then
      jq -nc --arg url "$smoke_base" '{seed_mining_nginx:"ok",mining_work:"ready",smoke_url:$url}'
      exit 0
    fi
    sleep 2
  done
done

echo "nginx updated but loopback mining work smoke check did not succeed" >&2
cat /tmp/seed-mining-work-smoke.json 2>/dev/null || true
exit 1
