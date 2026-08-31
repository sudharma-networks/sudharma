import assert from 'node:assert/strict';
import fs from 'node:fs';
import test from 'node:test';

// These workflows can mutate the public testnet, AWS infrastructure, or chain
// state. Some are not present on main yet; naming them here makes the guard
// effective as soon as a canonical-integration change restores one of them.
const mutatingWorkflows = [
  '.github/workflows/demand-miner-auto-deploy.yml',
  '.github/workflows/explorer-public-rpc-deploy.yml',
  '.github/workflows/explorer-seed-rpc-deploy.yml',
  '.github/workflows/faucet-diagnostics-auto-deploy.yml',
  '.github/workflows/faucet-enable-public.yml',
  '.github/workflows/faucet-fresh-start.yml',
  '.github/workflows/faucet-live-diagnostics.yml',
  '.github/workflows/faucet-prepared-payout-recovery.yml',
  '.github/workflows/faucet-recovery-monitor.yml',
  '.github/workflows/faucet-recovery-retry.yml',
  '.github/workflows/faucet-refill.yml',
  '.github/workflows/gpu-staging-aws-capability-preflight.yml',
  '.github/workflows/provision-faucet-endpoints.yml',
  '.github/workflows/faucet-payout-diagnostic.yml',
  '.github/workflows/inspect-faucet-vpc.yml',
  '.github/workflows/testnet-public-rpc.yml',
  '.github/workflows/verify-testnet-signed-transfer-once.yml',
];

function triggerBlock(source) {
  const match = /^on:\s*\n([\s\S]*?)(?=^[a-zA-Z][\w-]*:)/m.exec(source);
  assert.ok(match, 'workflow must have a top-level on block');
  return match[1];
}

function assertManualOnly(source, name) {
  const triggers = triggerBlock(source);
  assert.match(
    triggers,
    /^  workflow_dispatch:/m,
    `${name} must support manual dispatch`,
  );
  assert.doesNotMatch(
    triggers,
    /^  (push|schedule|workflow_run|workflow_call):/m,
    `${name} must not run automatically or transitively`,
  );
}

test('manual-only trigger policy rejects automatic and transitive triggers', () => {
  assertManualOnly('on:\n  workflow_dispatch:\n\njobs:\n', 'manual fixture');

  for (const trigger of ['push', 'schedule', 'workflow_run', 'workflow_call']) {
    assert.throws(
      () =>
        assertManualOnly(
          `on:\n  workflow_dispatch:\n  ${trigger}:\n\njobs:\n`,
          `${trigger} fixture`,
        ),
      /must not run automatically or transitively/,
    );
  }
});

test('public-testnet mutation workflows are manual-only when present', () => {
  for (const workflow of mutatingWorkflows) {
    if (!fs.existsSync(workflow)) {
      continue;
    }
    assertManualOnly(fs.readFileSync(workflow, 'utf8'), workflow);
  }
});

// Unauthenticated GitHub API calls share a 60-requests-per-hour IP budget on
// hosted runners, which fails CI with a 403 rather than a real defect.
test('workflow steps calling the GitHub API pass a token', () => {
  const workflowDir = '.github/workflows';
  if (!fs.existsSync(workflowDir)) return;

  for (const entry of fs.readdirSync(workflowDir)) {
    if (!/\.ya?ml$/.test(entry)) continue;
    const source = fs.readFileSync(`${workflowDir}/${entry}`, 'utf8');
    const steps = source.split(/\n\s*- (?=name:|uses:|run:)/);
    for (const step of steps) {
      if (!/\brun:[\s\S]*?(sync:github|gh api|gh release|gh run )/.test(step)) continue;
      assert.match(
        step,
        /(GH_TOKEN|GITHUB_TOKEN):/,
        `${workflowDir}/${entry} calls the GitHub API without passing a token`,
      );
    }
  }
});

test('Android CI cannot publish releases', () => {
  const workflow = '.github/workflows/android-wallet.yml';
  if (!fs.existsSync(workflow)) {
    return;
  }

  const source = fs.readFileSync(workflow, 'utf8');
  assert.doesNotMatch(source, /^\s*contents:\s*write\s*$/m);
  assert.doesNotMatch(source, /\bgh release (create|upload)\b/);
});
