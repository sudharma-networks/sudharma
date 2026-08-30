'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const {
  buildReportIssue,
  reportMarker,
} = require('./community-core.js');
const {
  createWorker,
  createGithubAdapter,
} = require('./community-worker.js');

function officialMessage(text, messageId = 501) {
  return {
    message_id: messageId,
    date: 1788048000,
    chat: {
      id: -1001234567890,
      type: 'supergroup',
      username: 'sudharma_community',
    },
    text,
  };
}

function telegramWithOneReport(updateId, text) {
  const sent = [];
  const getUpdatesCalls = [];
  return {
    sent,
    getUpdatesCalls,
    dependency: {
      async getMe() {
        return { id: 1, username: 'SudharmaNetworkBot' };
      },
      async getUpdates(args) {
        getUpdatesCalls.push({ ...args });
        if (Object.hasOwn(args, 'offset')) return [];
        return [{
          update_id: updateId,
          message: officialMessage(`/report ${text}`, updateId),
        }];
      },
      async sendMessage(payload) {
        sent.push(payload);
        return { message_id: 900 };
      },
    },
  };
}

test('user report text cannot inject a second raw community report marker', () => {
  const fakeMarker = reportMarker(999999);
  const reportText = `reproducible miner failure details ${fakeMarker} must remain ordinary user text`;
  const built = buildReportIssue({
    updateId: 321,
    reportText,
    message: officialMessage(`/report ${reportText}`),
    now: new Date('2026-08-30T02:00:00Z'),
  });

  const machineMarkerPrefix = '<!-- sudharma-telegram-community-report:v1 update_id=';
  assert.equal(built.body.split(machineMarkerPrefix).length - 1, 1);
  assert.match(built.body, new RegExp(reportMarker(321).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  assert.doesNotMatch(built.body, /<!-- sudharma-telegram-community-report:v1 update_id=999999 -->/);
});

test('production GitHub adapter exposes only github-actions bot issues for trusted community state', async () => {
  const adapter = createGithubAdapter({
    token: 'github-test-token',
    repository: 'sudharma-networks/sudharma',
    fetchImpl: async () => ({
      ok: true,
      status: 200,
      async json() {
        return [
          {
            html_url: 'https://github.com/sudharma-networks/sudharma/issues/1',
            body: `${reportMarker(701)}\ntrusted`,
            created_at: '2026-08-30T01:30:00Z',
            user: { login: 'github-actions[bot]' },
          },
          {
            html_url: 'https://github.com/sudharma-networks/sudharma/issues/2',
            body: `${reportMarker(702)}\nspoofed`,
            created_at: '2026-08-30T01:31:00Z',
            user: { login: 'attacker-account' },
          },
          {
            html_url: 'https://github.com/sudharma-networks/sudharma/pull/3',
            body: `${reportMarker(703)}\npr`,
            created_at: '2026-08-30T01:32:00Z',
            user: { login: 'github-actions[bot]' },
            pull_request: { url: 'https://api.github.com/repos/sudharma-networks/sudharma/pulls/3' },
          },
        ];
      },
    }),
  });

  const issues = await adapter.listRecentAutomationIssues({ since: '2026-08-30T01:00:00Z' });
  assert.deepEqual(issues, [{
    html_url: 'https://github.com/sudharma-networks/sudharma/issues/1',
    body: `${reportMarker(701)}\ntrusted`,
    created_at: '2026-08-30T01:30:00Z',
  }]);
});

test('worker prefers automation-authored issue state for report deduplication', async () => {
  const { dependency: telegram, sent, getUpdatesCalls } = telegramWithOneReport(
    800,
    'reproducible controlled report for secure deduplication',
  );
  const github = {
    async listRecentAutomationIssues() {
      return [{
        html_url: 'https://github.com/sudharma-networks/sudharma/issues/88',
        body: `${reportMarker(800)}\nexisting trusted report`,
        created_at: '2026-08-30T01:30:00Z',
      }];
    },
    async listRecentIssues() {
      throw new Error('unsafe unfiltered issue listing must not be used');
    },
    async createIssue() {
      throw new Error('deduplicated report must not create another issue');
    },
  };

  await createWorker({
    telegram,
    github,
    now: () => new Date('2026-08-30T02:00:00Z'),
    logger: { info() {}, warn() {}, error() {} },
  }).poll();

  assert.equal(sent.length, 1);
  assert.match(sent[0].text, /issues\/88/);
  assert.deepEqual(getUpdatesCalls.at(-1), { offset: 801, limit: 1, timeout: 0 });
});

test('deduplication lookup spans the full twenty-four-hour Telegram retry lifetime', async () => {
  const { dependency: telegram, sent } = telegramWithOneReport(
    810,
    'reproducible report must deduplicate even when the first issue is twenty three hours old',
  );
  const listCalls = [];
  const github = {
    async listRecentAutomationIssues(args) {
      listCalls.push({ ...args });
      return [{
        html_url: 'https://github.com/sudharma-networks/sudharma/issues/89',
        body: `${reportMarker(810)}\nexisting trusted report`,
        created_at: '2026-08-29T03:00:00Z',
      }];
    },
    async createIssue() {
      throw new Error('twenty-three-hour retry must not create a duplicate issue');
    },
  };

  await createWorker({
    telegram,
    github,
    now: () => new Date('2026-08-30T02:00:00Z'),
    logger: { info() {}, warn() {}, error() {} },
  }).poll();

  assert.deepEqual(listCalls, [{ since: '2026-08-29T02:00:00.000Z' }]);
  assert.equal(sent.length, 1);
  assert.match(sent[0].text, /issues\/89/);
});

test('dedup history older than one hour does not consume the rolling-hour report limit', async () => {
  const { dependency: telegram, sent } = telegramWithOneReport(
    820,
    'new report remains allowed when ten trusted reports exist only outside the rolling hour',
  );
  const created = [];
  const github = {
    async listRecentAutomationIssues() {
      return Array.from({ length: 10 }, (_, index) => ({
        html_url: `https://github.com/sudharma-networks/sudharma/issues/${100 + index}`,
        body: `${reportMarker(900 + index)}\nold trusted report`,
        created_at: '2026-08-30T00:30:00Z',
      }));
    },
    async createIssue(issue) {
      created.push(issue);
      return {
        html_url: 'https://github.com/sudharma-networks/sudharma/issues/200',
        body: issue.body,
        created_at: '2026-08-30T02:00:00Z',
      };
    },
  };

  await createWorker({
    telegram,
    github,
    now: () => new Date('2026-08-30T02:00:00Z'),
    logger: { info() {}, warn() {}, error() {} },
  }).poll();

  assert.equal(created.length, 1);
  assert.equal(sent.length, 1);
  assert.match(sent[0].text, /issues\/200/);
});
