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

def mining_allowlist_ready(text: str) -> bool:
    return "mining/work" in text and "mining/submit" in text

def patch_regex_allowlist(text: str) -> tuple[str, bool]:
    pattern = re.compile(
        r"(location\s+~\s+\^/v1/\()([^)]+)(\)[^{]*\{[^}]*?proxy_pass\s+[^;]+;[^}]*\})",
        re.S,
    )
    match = pattern.search(text)
    if not match or mining_allowlist_ready(match.group(2)):
        return text, False
    updated_group = match.group(2).rstrip("|") + "|mining/work|mining/submit|"
    return text[:match.start(2)] + updated_group + text[match.end(2):], True

def patch_exact_locations(text: str, upstream: str) -> tuple[str, bool]:
    if mining_allowlist_ready(text):
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
    if mining_allowlist_ready(text):
        continue
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

nginx_smoke_url="${SEED_RPC_SMOKE_URL:-}"
if [ -z "$nginx_smoke_url" ] && [ -n "${SEED_PRIVATE_IP:-}" ]; then
  nginx_smoke_url="http://${SEED_PRIVATE_IP}:29100"
fi
if [ -z "$nginx_smoke_url" ]; then
  echo "SEED_RPC_SMOKE_URL or SEED_PRIVATE_IP is required for nginx mining smoke" >&2
  exit 1
fi
nginx_smoke_url="${nginx_smoke_url%/}"

for attempt in $(seq 1 12); do
  status_code="$(curl -sS -o /tmp/seed-mining-work-smoke.json -w '%{http_code}' \
    -X POST "${nginx_smoke_url}/v1/mining/work" \
    -H 'content-type: application/json' \
    --data '{"address":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}')"
  if [ "$status_code" = '200' ] && jq -e '.algorithm and (.block or (.header_prefix and .target))' /tmp/seed-mining-work-smoke.json >/dev/null 2>&1; then
    jq -nc --arg url "$nginx_smoke_url" '{seed_mining_nginx:"ok",mining_work:"ready",smoke_url:$url}'
    exit 0
  fi
  echo "nginx mining smoke attempt $attempt: HTTP $status_code" >&2
  head -c 400 /tmp/seed-mining-work-smoke.json 2>/dev/null >&2 || true
  echo >&2
  sleep 2
done

echo "seed nginx on ${nginx_smoke_url} did not serve POST /v1/mining/work" >&2
if command -v nginx >/dev/null 2>&1; then
  nginx -T 2>/dev/null | grep -E 'listen|/v1/mining|/v1/status|29100' | head -80 >&2 || true
fi
exit 1
