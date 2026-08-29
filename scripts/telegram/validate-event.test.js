const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');

const VALID_BODY = `<!-- sudharma-telegram-bridge:v1 -->\nTELEGRAM_MESSAGE_BEGIN\nHello from Sudharma\n\nSecond line.\nTELEGRAM_MESSAGE_END`;
const SCRIPT = path.join(__dirname, 'validate-event.js');

function runValidator({ mode = 'dry-run', association = 'OWNER', label, body = VALID_BODY, pullRequest = false } = {}) {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'sudharma-telegram-'));
  const eventPath = path.join(tempDir, 'event.json');
  const outputPath = path.join(tempDir, 'message.txt');
  const expectedLabel = label || (mode === 'publish' ? 'telegram:publish-approved' : 'telegram:dry-run');
  const issue = { number: 77, body, author_association: association };
  if (pullRequest) issue.pull_request = { url: 'https://example.invalid/pr/77' };
  fs.writeFileSync(eventPath, JSON.stringify({ action: 'labeled', issue, label: { name: expectedLabel } }));

  const result = spawnSync(process.execPath, [SCRIPT], {
    encoding: 'utf8',
    env: {
      ...process.env,
      GITHUB_EVENT_PATH: eventPath,
      TELEGRAM_MODE: mode,
      TELEGRAM_MESSAGE_FILE: outputPath,
    },
  });

  return {
    ...result,
    outputPath,
    message: fs.existsSync(outputPath) ? fs.readFileSync(outputPath, 'utf8') : null,
  };
}

test('dry-run accepts trusted exact label and writes only validated message', () => {
  const result = runValidator();
  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.message, 'Hello from Sudharma\n\nSecond line.');
});

test('publish accepts trusted exact publish label', () => {
  const result = runValidator({ mode: 'publish' });
  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.message, 'Hello from Sudharma\n\nSecond line.');
});

test('rejects untrusted issue association', () => {
  const result = runValidator({ association: 'CONTRIBUTOR' });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /trusted/i);
  assert.equal(result.message, null);
});

test('rejects a label that does not exactly match the selected mode', () => {
  const result = runValidator({ mode: 'publish', label: 'telegram:dry-run' });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /label/i);
  assert.equal(result.message, null);
});

test('rejects malformed command body', () => {
  const result = runValidator({ body: 'hello without markers' });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /marker/i);
  assert.equal(result.message, null);
});

test('rejects pull-request-shaped issue payload', () => {
  const result = runValidator({ pullRequest: true });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /pull request/i);
  assert.equal(result.message, null);
});

test('rejects unknown mode', () => {
  const result = runValidator({ mode: 'delete', label: 'telegram:publish-approved' });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /mode/i);
  assert.equal(result.message, null);
});

test('workflow is least privilege and separates dry-run from publish', () => {
  const workflowPath = path.resolve(__dirname, '../../.github/workflows/telegram-publish.yml');
  const workflow = fs.readFileSync(workflowPath, 'utf8');

  assert.match(workflow, /contents:\s*read/);
  assert.match(workflow, /issues:\s*write/);
  assert.doesNotMatch(workflow, /contents:\s*write/);
  assert.match(workflow, /group:\s*telegram-issue-\$\{\{ github\.event\.issue\.number \}\}/);
  assert.match(workflow, /telegram:dry-run/);
  assert.match(workflow, /telegram:publish-approved/);
  assert.doesNotMatch(workflow, /pull_request_target/);
  assert.doesNotMatch(workflow, /aws-actions|amazonaws|AWS_/i);

  const dryStart = workflow.indexOf('  dry-run:');
  const publishStart = workflow.indexOf('  publish:');
  assert.ok(dryStart !== -1 && publishStart > dryStart, 'separate dry-run and publish jobs are required');
  const drySection = workflow.slice(dryStart, publishStart);
  const publishSection = workflow.slice(publishStart);
  assert.doesNotMatch(drySection, /TELEGRAM_BOT_TOKEN|api\.telegram\.org/);
  assert.match(publishSection, /TELEGRAM_BOT_TOKEN/);
  assert.match(publishSection, /api\.telegram\.org/);
  assert.match(publishSection, /link_preview_options/);
  assert.doesNotMatch(publishSection, /disable_web_page_preview/);
});
