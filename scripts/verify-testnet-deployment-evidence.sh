#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <deployment-evidence.json> [expected-rc-commit]" >&2
  exit 2
fi

EVIDENCE="$1"
EXPECTED_RC="${2:-}"

if [[ ! -f "$EVIDENCE" ]]; then
  echo "missing evidence file: $EVIDENCE" >&2
  exit 1
fi

python3 - "$EVIDENCE" "$EXPECTED_RC" <<'PY'
import json, re, sys

path, expected_rc = sys.argv[1], sys.argv[2]
with open(path, encoding='utf-8') as handle:
    data = json.load(handle)

if data.get('kind') != 'sudharma-testnet-deployment-evidence':
    raise SystemExit('evidence kind must be sudharma-testnet-deployment-evidence')

raw = json.dumps(data)
if 'REPLACE_WITH_' in raw:
    raise SystemExit('evidence still contains REPLACE_WITH_ placeholders')

commit = str(data.get('rc_candidate_commit', '')).strip()
if not re.fullmatch(r'[0-9a-f]{40}', commit):
    raise SystemExit('rc_candidate_commit must be a 40-character git commit SHA')

if expected_rc and commit != expected_rc:
    raise SystemExit(f'rc_candidate_commit mismatch: evidence={commit} expected={expected_rc}')

required_paths = [
    ('recorded_at', str),
    ('components.seed1.commit_or_artifact_sha256', str),
    ('components.seed2.commit_or_artifact_sha256', str),
    ('components.public_rpc_lambda.code_sha256', str),
    ('public_rpc_smoke.rpc_base_url', str),
    ('public_rpc_smoke.collected_at', str),
    ('operator_signoff.reviewed_by', str),
]

optional_component_groups = {
    'components.demand_miner_seed1': ['commit_or_artifact_sha256'],
    'components.demand_miner_seed2': ['commit_or_artifact_sha256'],
    'components.website': ['build_id', 'deployment_url'],
    'components.android_wallet': ['tag', 'commit', 'checksum_sha256'],
}

def get_path(obj, dotted):
    cur = obj
    for part in dotted.split('.'):
        if not isinstance(cur, dict) or part not in cur:
            return None
        cur = cur[part]
    return cur

def require_deferred_notes(component_path):
    component = get_path(data, component_path)
    if not isinstance(component, dict):
        raise SystemExit(f'missing component object: {component_path}')
    if component.get('deferred') is not True:
        return False
    notes = component.get('notes')
    if not isinstance(notes, str) or not notes.strip():
        raise SystemExit(f'{component_path}.notes is required when deferred=true')
    return True

for dotted, expected_type in required_paths:
    value = get_path(data, dotted)
    if value is None or value == '':
        raise SystemExit(f'missing required evidence field: {dotted}')
    if not isinstance(value, expected_type):
        raise SystemExit(f'field {dotted} must be {expected_type.__name__}')

for component_path, fields in optional_component_groups.items():
    if require_deferred_notes(component_path):
        continue
    for field in fields:
        dotted = f'{component_path}.{field}'
        value = get_path(data, dotted)
        if value is None or value == '':
            raise SystemExit(f'missing required evidence field: {dotted} (or set {component_path}.deferred=true with notes)')
        if not isinstance(value, str):
            raise SystemExit(f'field {dotted} must be str')

smoke = data.get('public_rpc_smoke', {})
for key in ('ready', 'status', 'explorer_status', 'faucet_health'):
    if not isinstance(smoke.get(key), dict):
        raise SystemExit(f'public_rpc_smoke.{key} must be an object')

if smoke.get('status', {}).get('network') != 'sudharma':
    raise SystemExit('public_rpc_smoke.status.network must be sudharma')

if smoke.get('explorer_status', {}).get('network') != 'sudharma':
    raise SystemExit('public_rpc_smoke.explorer_status.network must be sudharma')

if not isinstance(smoke.get('visitor_total'), int) or smoke['visitor_total'] < 0:
    raise SystemExit('public_rpc_smoke.visitor_total must be a non-negative integer')

print('PASS: deployment evidence is structurally valid and matches RC expectations')
PY
