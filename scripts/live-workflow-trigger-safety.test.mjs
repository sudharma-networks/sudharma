import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const mutatingWorkflows = [
  '.github/workflows/demand-miner-auto-deploy.yml',
  '.github/workflows/explorer-public-rpc-deploy.yml',
  '.github/workflows/explorer-seed-rpc-deploy.yml',
  '.github/workflows/faucet-diagnostics-auto-deploy.yml',
  '.github/workflows/faucet-enable-public.yml',
  '.github/workflows/faucet-fresh-start.yml',
  '.github/workflows/faucet-prepared-payout-recovery.yml',
  '.github/workflows/faucet-recovery-retry.yml',
  '.github/workflows/faucet-refill.yml',
  '.github/workflows/testnet-public-rpc.yml',
];

function triggerBlock(source) {
  const match = /^on:\n([\s\S]*?)(?=^[a-zA-Z][\w-]*:)/m.exec(source);
  assert.ok(match, 'workflow must have a top-level on block');
  return match[1];
}

test('AWS and chain mutating workflows are manual-only', () => {
  for (const workflow of mutatingWorkflows) {
    const triggers = triggerBlock(fs.readFileSync(workflow, 'utf8'));
    assert.match(triggers, /^  workflow_dispatch:/m, `${workflow} must support manual dispatch`);
    assert.doesNotMatch(triggers, /^  (push|schedule|workflow_run|workflow_call):/m, `${workflow} must not run automatically or transitively`);
  }
});
