import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';

const repoRoot = path.resolve(path.dirname(new URL(import.meta.url).pathname), '..');
const verifyScript = path.join(repoRoot, 'scripts/verify-testnet-deployment-evidence.sh');

function runVerify(evidencePath, expectedRc = '') {
  const args = [verifyScript, evidencePath];
  if (expectedRc) args.push(expectedRc);
  return spawnSync('bash', args, { encoding: 'utf8' });
}

test('verify rejects template placeholders', () => {
  const template = path.join(repoRoot, 'deployment/testnet/deployment-evidence.template.json');
  const result = runVerify(template);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr + result.stdout, /REPLACE_WITH_/);
});

test('verify accepts complete evidence fixture', () => {
  const fixture = {
    kind: 'sudharma-testnet-deployment-evidence',
    recorded_at: '2026-08-31T12:00:00Z',
    rc_candidate_commit: '5f9258918fb301009a4e37ceb3f522906a8fd699',
    components: {
      seed1: { commit_or_artifact_sha256: 'a'.repeat(64), service_unit: 'sudharma-testnet.service', observed_height: 12 },
      seed2: { commit_or_artifact_sha256: 'b'.repeat(64), service_unit: 'sudharma-testnet.service', observed_height: 12 },
      public_rpc_lambda: { function_name: 'Sudharma-Testnet-Wallet-Proxy', code_sha256: 'c'.repeat(64), faucet_enabled: false },
      demand_miner_seed1: { commit_or_artifact_sha256: 'd'.repeat(64) },
      demand_miner_seed2: { commit_or_artifact_sha256: 'e'.repeat(64) },
      website: { build_id: 'web-build-1', deployment_url: 'https://example.com' },
      android_wallet: { tag: 'v0.1.0-testnet', commit: '5f9258918fb301009a4e37ceb3f522906a8fd699', checksum_sha256: 'f'.repeat(64) },
    },
    public_rpc_smoke: {
      rpc_base_url: 'https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com',
      collected_at: '2026-08-31T12:00:00Z',
      ready: { status: 'ready' },
      status: { network: 'sudharma', height: 12 },
      explorer_status: { network: 'sudharma', height: 12 },
      faucet_health: { ready: true },
      visitor_total: 3,
    },
    operator_signoff: { reviewed_by: 'operator@example.com', notes: 'manual-only deploy complete' },
  };

  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'sudharma-evidence-'));
  const file = path.join(tempDir, 'evidence.json');
  fs.writeFileSync(file, JSON.stringify(fixture, null, 2));
  const result = runVerify(file, fixture.rc_candidate_commit);
  assert.equal(result.status, 0, result.stderr || result.stdout);
});
