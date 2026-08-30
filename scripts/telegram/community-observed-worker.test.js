'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const { instrumentDependency } = require('./community-observed-worker.js');

test('observability wrapper logs only safe operation names and preserves successful results', async () => {
  const events = [];
  const dependency = {
    async createIssue(payload) {
      assert.deepEqual(payload, { secretReportText: 'must never appear in logs' });
      return { html_url: 'https://example.invalid/issues/1' };
    },
  };
  const logger = {
    info(message) { events.push(String(message)); },
    error(message) { events.push(String(message)); },
  };

  const wrapped = instrumentDependency('github', dependency, ['createIssue'], logger);
  const result = await wrapped.createIssue({ secretReportText: 'must never appear in logs' });

  assert.equal(result.html_url, 'https://example.invalid/issues/1');
  assert.deepEqual(events, ['Telegram community boundary ok: github.createIssue']);
  assert.doesNotMatch(events.join('\n'), /secretReportText|must never appear/);
});

test('observability wrapper logs a safe failure boundary without leaking the thrown error message', async () => {
  const events = [];
  const secret = 'TOKEN_OR_REPORT_TEXT_MUST_NOT_LEAK';
  const failure = new Error(`synthetic failure ${secret}`);
  const dependency = {
    async listRecentAutomationIssues() {
      throw failure;
    },
  };
  const logger = {
    info(message) { events.push(String(message)); },
    error(message) { events.push(String(message)); },
  };

  const wrapped = instrumentDependency('github', dependency, ['listRecentAutomationIssues'], logger);
  await assert.rejects(() => wrapped.listRecentAutomationIssues({ since: '2026-08-29T00:00:00Z' }), failure);

  assert.deepEqual(events, ['Telegram community boundary failed: github.listRecentAutomationIssues']);
  assert.doesNotMatch(events.join('\n'), new RegExp(secret));
});

test('observability wrapper may log only a pre-sanitized diagnostic code from a failure', async () => {
  const events = [];
  const secret = 'RAW_TELEGRAM_DESCRIPTION_MUST_NOT_LEAK';
  const failure = new Error(`Bad Request: ${secret}`);
  failure.diagnosticCode = 'telegram-api-400-reply-target-missing';
  const dependency = {
    async sendMessage() {
      throw failure;
    },
  };
  const logger = {
    info(message) { events.push(String(message)); },
    error(message) { events.push(String(message)); },
  };

  const wrapped = instrumentDependency('telegram', dependency, ['sendMessage'], logger);
  await assert.rejects(() => wrapped.sendMessage({ text: secret }), failure);

  assert.deepEqual(events, [
    'Telegram community boundary failed: telegram.sendMessage code=telegram-api-400-reply-target-missing',
  ]);
  assert.doesNotMatch(events.join('\n'), new RegExp(secret));
});

test('observability wrapper rejects unsafe diagnostic-code text from logs', async () => {
  const events = [];
  const secret = 'LEAK_ME_NOT';
  const failure = new Error('synthetic');
  failure.diagnosticCode = `telegram-api-400-${secret}`;
  const dependency = {
    async sendMessage() {
      throw failure;
    },
  };
  const logger = {
    info(message) { events.push(String(message)); },
    error(message) { events.push(String(message)); },
  };

  const wrapped = instrumentDependency('telegram', dependency, ['sendMessage'], logger);
  await assert.rejects(() => wrapped.sendMessage({}), failure);

  assert.deepEqual(events, ['Telegram community boundary failed: telegram.sendMessage']);
  assert.doesNotMatch(events.join('\n'), new RegExp(secret));
});
