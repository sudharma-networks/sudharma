'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const repoRoot = path.resolve(__dirname, '..', '..');
const communityWorkflowPath = path.join(repoRoot, '.github', 'workflows', 'telegram-community.yml');
const publishWorkflowPath = path.join(repoRoot, '.github', 'workflows', 'telegram-publish.yml');

test('community workflow is manual-only during initial rollout', () => {
  const workflow = fs.readFileSync(communityWorkflowPath, 'utf8');
  assert.match(workflow, /workflow_dispatch:/);
  assert.match(workflow, /mode:/);
  assert.match(workflow, /bootstrap/);
  assert.match(workflow, /poll/);
  assert.doesNotMatch(workflow, /\bschedule:/);
  assert.doesNotMatch(workflow, /cron:/);
});

test('community workflow uses least privilege and serialized polling', () => {
  const workflow = fs.readFileSync(communityWorkflowPath, 'utf8');
  assert.match(workflow, /permissions:[\s\S]*contents:\s*read/);
  assert.match(workflow, /permissions:[\s\S]*issues:\s*write/);
  assert.doesNotMatch(workflow, /contents:\s*write/);
  assert.match(workflow, /concurrency:[\s\S]*group:\s*telegram-community-poller/);
  assert.match(workflow, /cancel-in-progress:\s*false/);
});

test('community workflow bounds each serialized operation with a short timeout', () => {
  const workflow = fs.readFileSync(communityWorkflowPath, 'utf8');
  assert.match(workflow, /jobs:[\s\S]*community:[\s\S]*timeout-minutes:\s*5/);
});

test('community workflow reuses the protected Telegram environment secret and GitHub token', () => {
  const workflow = fs.readFileSync(communityWorkflowPath, 'utf8');
  assert.match(workflow, /environment:\s*telegram-publishing/);
  assert.match(workflow, /TELEGRAM_BOT_TOKEN:\s*\$\{\{\s*secrets\.TELEGRAM_BOT_TOKEN\s*\}\}/);
  assert.match(workflow, /GITHUB_TOKEN:\s*\$\{\{\s*github\.token\s*\}\}/);
  assert.match(workflow, /GITHUB_REPOSITORY:\s*\$\{\{\s*github\.repository\s*\}\}/);
  assert.match(workflow, /COMMUNITY_MODE:/);
  assert.match(workflow, /node scripts\/telegram\/community-observed-worker\.js/);
  assert.doesNotMatch(workflow, /echo[^\n]*TELEGRAM_BOT_TOKEN|print[^\n]*TELEGRAM_BOT_TOKEN/i);
});

test('community workflow does not introduce AWS or a second Telegram secret', () => {
  const workflow = fs.readFileSync(communityWorkflowPath, 'utf8');
  assert.doesNotMatch(workflow, /aws-actions|amazonaws|AWS_/i);
  const secretRefs = [...workflow.matchAll(/secrets\.([A-Z0-9_]+)/g)].map((match) => match[1]);
  assert.deepEqual([...new Set(secretRefs)], ['TELEGRAM_BOT_TOKEN']);
});

test('Phase 1 announcement workflow remains present with its existing authorization labels and Telegram publish path', () => {
  const workflow = fs.readFileSync(publishWorkflowPath, 'utf8');
  assert.match(workflow, /telegram:dry-run/);
  assert.match(workflow, /telegram:publish-approved/);
  assert.match(workflow, /telegram-publishing/);
  assert.match(workflow, /api\.telegram\.org/);
  assert.match(workflow, /TELEGRAM_TRIGGER_ACTOR:\s*\$\{\{\s*github\.actor\s*\}\}/);
});
