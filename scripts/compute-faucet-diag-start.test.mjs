import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const script = path.join(path.dirname(fileURLToPath(import.meta.url)), 'compute-faucet-diag-start.mjs');

test('diagnostic window starts two minutes before prepared_at', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'faucet-diag-start-'));
  const itemPath = path.join(dir, 'item.json');
  const envPath = path.join(dir, 'github.env');
  fs.writeFileSync(itemPath, JSON.stringify({
    initial_prepared_at: '1788094607291',
    initial_reserved_at: '1788094606593',
  }));
  const result = spawnSync(process.execPath, [script, itemPath], {
    env: { ...process.env, DIAG_START_MS: '1', GITHUB_ENV: envPath },
    encoding: 'utf8',
  });
  assert.equal(result.status, 0, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout), {
    diag_start_ms: 1788094487291,
    used_prepared_at: true,
  });
  assert.equal(fs.readFileSync(envPath, 'utf8'), 'DIAG_START_MS=1788094487291\n');
});
