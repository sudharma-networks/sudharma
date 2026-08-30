#!/usr/bin/env sh
set -eu

: "${EXPECTED_OLD_SHA:?EXPECTED_OLD_SHA is required}"
: "${EXPECTED_NEW_SHA:?EXPECTED_NEW_SHA is required}"
: "${EXPECTED_NODE_ID:?EXPECTED_NODE_ID is required}"
: "${PRIVATE_IP:?PRIVATE_IP is required}"
: "${ARTIFACT_URL_B64:?ARTIFACT_URL_B64 is required}"

BINARY=/usr/local/bin/sudharma-rpcd
SERVICE=sudharma.service
RPC_BASE=http://127.0.0.1:28545

fail() {
  echo "rollout failed: $*" >&2
  exit 1
}

[ "$(id -u)" = '0' ] || fail 'SSM command is not running as root'

for tool in curl python3 sha256sum systemctl base64 install mktemp awk cut; do
  command -v "$tool" >/dev/null 2>&1 || fail "required tool missing: $tool"
done

current_sha="$(sha256sum /usr/local/bin/sudharma-rpcd | awk '{print $1}')"
case "$current_sha" in
  "$EXPECTED_OLD_SHA"|"$EXPECTED_NEW_SHA") ;;
  *) fail "unexpected current binary sha: $current_sha" ;;
esac

systemctl is-active --quiet "$SERVICE" || fail 'sudharma.service is not active before rollout'
before_status="$(curl -fsS "$RPC_BASE/v1/status")" || fail 'status endpoint unavailable before rollout'
before_node="$(printf '%s' "$before_status" | python3 -c 'import json,sys; print(json.load(sys.stdin)["node_id"])')"
before_height="$(printf '%s' "$before_status" | python3 -c 'import json,sys; print(json.load(sys.stdin)["height"])')"
before_supply="$(printf '%s' "$before_status" | python3 -c 'import json,sys; print(json.load(sys.stdin)["issued_supply"])')"
[ "$before_node" = "$EXPECTED_NODE_ID" ] || fail "unexpected node id before rollout: $before_node"

backup="/usr/local/bin/sudharma-rpcd.rollback-${EXPECTED_OLD_SHA}"
changed=0
tmp=''

cleanup() {
  if [ -n "$tmp" ] && [ -d "$tmp" ]; then
    rm -rf "$tmp"
  fi
}
trap cleanup EXIT INT TERM

rollback() {
  if [ "$changed" = '1' ] && [ -f "$backup" ]; then
    echo 'Rolling back node binary.' >&2
    install -m 0755 "$backup" "$BINARY"
    systemctl restart sudharma.service || true
    attempt=0
    while [ "$attempt" -lt 30 ]; do
      if systemctl is-active --quiet "$SERVICE" && curl -fsS "$RPC_BASE/ready" >/dev/null 2>&1; then
        return 0
      fi
      attempt=$((attempt + 1))
      sleep 2
    done
  fi
  return 0
}

fail_after_change() {
  message="$1"
  echo "$message" >&2
  rollback
  exit 1
}

if [ "$current_sha" = "$EXPECTED_OLD_SHA" ]; then
  tmp="$(mktemp -d /tmp/sudharma-explorer-rollout.XXXXXX)"
  download_url="$(printf '%s' "$ARTIFACT_URL_B64" | base64 -d)"
  [ -n "$download_url" ] || fail 'artifact URL decoded empty'

  curl -fL --retry 3 --connect-timeout 10 --max-time 120 \
    -o "$tmp/artifact.zip" "$download_url" || fail 'artifact download failed'
  mkdir -p "$tmp/extracted"
  python3 -m zipfile -e "$tmp/artifact.zip" "$tmp/extracted" || fail 'artifact extraction failed'
  candidate="$tmp/extracted/sudharma-rpcd"
  [ -f "$candidate" ] || fail 'artifact did not contain sudharma-rpcd'

  candidate_sha="$(sha256sum "$candidate" | awk '{print $1}')"
  [ "$candidate_sha" = "$EXPECTED_NEW_SHA" ] || fail "candidate sha mismatch: $candidate_sha"

  if [ ! -f "$backup" ]; then
    cp -a "$BINARY" "$backup"
  fi
  backup_sha="$(sha256sum "$backup" | awk '{print $1}')"
  [ "$backup_sha" = "$EXPECTED_OLD_SHA" ] || fail "rollback backup sha mismatch: $backup_sha"

  install -m 0755 "$candidate" /usr/local/bin/sudharma-rpcd.next
  mv /usr/local/bin/sudharma-rpcd.next "$BINARY"
  changed=1
  systemctl restart sudharma.service || fail_after_change 'systemctl restart failed'
fi

ready=0
attempt=0
while [ "$attempt" -lt 45 ]; do
  if systemctl is-active --quiet "$SERVICE" && curl -fsS "$RPC_BASE/ready" >/dev/null 2>&1; then
    ready=1
    break
  fi
  attempt=$((attempt + 1))
  sleep 2
done
[ "$ready" = '1' ] || fail_after_change 'node did not become ready after rollout'

after_status=''
after_node=''
after_height='0'
after_supply='0'
peers='0'
attempt=0
while [ "$attempt" -lt 45 ]; do
  after_status="$(curl -fsS "$RPC_BASE/v1/status" || true)"
  if [ -n "$after_status" ]; then
    after_node="$(printf '%s' "$after_status" | python3 -c 'import json,sys; print(json.load(sys.stdin)["node_id"])' 2>/dev/null || true)"
    after_height="$(printf '%s' "$after_status" | python3 -c 'import json,sys; print(json.load(sys.stdin)["height"])' 2>/dev/null || echo 0)"
    after_supply="$(printf '%s' "$after_status" | python3 -c 'import json,sys; print(json.load(sys.stdin)["issued_supply"])' 2>/dev/null || echo 0)"
    peers="$(printf '%s' "$after_status" | python3 -c 'import json,sys; print(json.load(sys.stdin)["peers"])' 2>/dev/null || echo 0)"
    if [ "$after_node" = "$EXPECTED_NODE_ID" ] \
      && [ "$after_height" -ge "$before_height" ] \
      && [ "$after_supply" -ge "$before_supply" ] \
      && [ "$peers" -ge 1 ]; then
      break
    fi
  fi
  attempt=$((attempt + 1))
  sleep 2
done

[ -n "$after_status" ] || fail_after_change 'status endpoint unavailable after rollout'
[ "$after_node" = "$EXPECTED_NODE_ID" ] || fail_after_change 'node id changed after rollout'
[ "$after_height" -ge "$before_height" ] || fail_after_change 'chain height regressed after rollout'
[ "$after_supply" -ge "$before_supply" ] || fail_after_change 'issued supply regressed after rollout'
[ "$peers" -ge 1 ] || fail_after_change 'peer did not reconnect after rollout'

explorer_status="$(curl -fsS "$RPC_BASE/v1/explorer/status" || true)"
[ -n "$explorer_status" ] || fail_after_change 'direct explorer status is unavailable'
explorer_network="$(printf '%s' "$explorer_status" | python3 -c 'import json,sys; print(json.load(sys.stdin)["network"])' 2>/dev/null || true)"
explorer_height="$(printf '%s' "$explorer_status" | python3 -c 'import json,sys; print(json.load(sys.stdin)["height"])' 2>/dev/null || echo 0)"
[ "$explorer_network" = 'sudharma' ] || fail_after_change 'unexpected explorer network'
[ "$explorer_height" -ge "$before_height" ] || fail_after_change 'explorer height regressed'

bridge_status="$(curl -fsS "http://${PRIVATE_IP}:29100/v1/explorer/status" || true)"
[ -n "$bridge_status" ] || fail_after_change 'private nginx explorer bridge is unavailable'
bridge_network="$(printf '%s' "$bridge_status" | python3 -c 'import json,sys; print(json.load(sys.stdin)["network"])' 2>/dev/null || true)"
[ "$bridge_network" = 'sudharma' ] || fail_after_change 'private nginx explorer bridge returned unexpected network'

installed_sha="$(sha256sum /usr/local/bin/sudharma-rpcd | awk '{print $1}')"
[ "$installed_sha" = "$EXPECTED_NEW_SHA" ] || fail_after_change 'installed binary sha does not match pinned build'

echo "rollout_ok node=$EXPECTED_NODE_ID height=$after_height peers=$peers binary_sha=$installed_sha"
