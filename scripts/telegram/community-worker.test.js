'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const {
  createWorker,
  createTelegramAdapter,
  createGithubAdapter,
} = require('./community-worker.js');
const { reportMarker } = require('./community-core.js');

function official(text, messageId = 44, extra = {}) {
  return {
    message_id: messageId,
    date: 1788048000,
    chat: {
      id: -1001234567890,
      type: 'supergroup',
      username: 'sudharma_community',
    },
    text,
    ...extra,
  };
}

function otherGroup(text, messageId = 43) {
  return {
    ...official(text, messageId),
    chat: { id: -100999, type: 'supergroup', username: 'other_group' },
  };
}

function update(updateId, message) {
  return { update_id: updateId, message };
}

function fakeTelegram({ updates = [], failReplyMessageIds = [], webhookUrl = '' } = {}) {
  const state = {
    getUpdatesCalls: [],
    sent: [],
    deletedWebhook: [],
  };
  const failSet = new Set(failReplyMessageIds);

  return {
    state,
    async getMe() {
      return { id: 12345, username: 'SudharmaNetworkBot' };
    },
    async getWebhookInfo() {
      return { url: webhookUrl };
    },
    async deleteWebhook(args) {
      state.deletedWebhook.push(args);
      return true;
    },
    async getUpdates(args) {
      state.getUpdatesCalls.push({ ...args });
      if (Object.hasOwn(args, 'offset')) return [];
      return updates;
    },
    async sendMessage(payload) {
      state.sent.push(structuredClone(payload));
      const repliedTo = payload.reply_parameters && payload.reply_parameters.message_id;
      if (failSet.has(repliedTo)) {
        throw new Error(`synthetic send failure for message ${repliedTo}`);
      }
      return { message_id: 9000 + state.sent.length };
    },
  };
}

function fakeGithub({
  recentIssues = [],
  createdUrl = 'https://github.com/sudharma-networks/sudharma/issues/999',
  failCreate = false,
} = {}) {
  const state = {
    listCalls: [],
    created: [],
  };

  return {
    state,
    async listRecentIssues(args) {
      state.listCalls.push({ ...args });
      return structuredClone(recentIssues);
    },
    async createIssue(issue) {
      state.created.push(structuredClone(issue));
      if (failCreate) throw new Error('synthetic GitHub create failure');
      return {
        html_url: createdUrl,
        title: issue.title,
        body: issue.body,
        created_at: '2026-08-30T02:00:00Z',
      };
    },
  };
}

const fixedNow = () => new Date('2026-08-30T02:00:00Z');
const silentLogger = { info() {}, warn() {}, error() {} };

test('poll acknowledges an ignored update and a successful help command through one contiguous prefix', async () => {
  const telegram = fakeTelegram({
    updates: [
      update(10, otherGroup('/help', 10)),
      update(11, official('/help', 11)),
    ],
  });

  await createWorker({ telegram, github: fakeGithub(), now: fixedNow, logger: silentLogger }).poll();

  assert.equal(telegram.state.sent.length, 1);
  assert.match(telegram.state.sent[0].text, /\/report/);
  assert.deepEqual(telegram.state.getUpdatesCalls, [
    { limit: 100, timeout: 0 },
    { offset: 12, limit: 1, timeout: 0 },
  ]);
});

test('poll processes updates in ascending update_id order', async () => {
  const telegram = fakeTelegram({
    updates: [
      update(30, official('/rules', 30)),
      update(20, official('/help', 20)),
    ],
  });

  await createWorker({ telegram, github: fakeGithub(), now: fixedNow, logger: silentLogger }).poll();

  assert.deepEqual(
    telegram.state.sent.map((payload) => payload.reply_parameters.message_id),
    [20, 30],
  );
  assert.deepEqual(telegram.state.getUpdatesCalls.at(-1), { offset: 31, limit: 1, timeout: 0 });
});

test('a failed actionable reply stops later processing and acknowledges only the prior contiguous prefix', async () => {
  const telegram = fakeTelegram({
    updates: [
      update(10, otherGroup('/help', 10)),
      update(11, official('/help', 11)),
      update(12, official('/rules', 12)),
    ],
    failReplyMessageIds: [11],
  });

  await createWorker({ telegram, github: fakeGithub(), now: fixedNow, logger: silentLogger }).poll();

  assert.deepEqual(
    telegram.state.sent.map((payload) => payload.reply_parameters.message_id),
    [11],
  );
  assert.deepEqual(telegram.state.getUpdatesCalls.at(-1), { offset: 11, limit: 1, timeout: 0 });
});

test('failure on the first actionable update sends no acknowledgement offset', async () => {
  const telegram = fakeTelegram({
    updates: [update(50, official('/help', 50))],
    failReplyMessageIds: [50],
  });

  await createWorker({ telegram, github: fakeGithub(), now: fixedNow, logger: silentLogger }).poll();

  assert.deepEqual(telegram.state.getUpdatesCalls, [{ limit: 100, timeout: 0 }]);
});

test('command replies are plain text, reply to the source message, disable previews, and preserve forum topic', async () => {
  const telegram = fakeTelegram({
    updates: [update(60, official('/testnet', 60, { message_thread_id: 777 }))],
  });

  await createWorker({ telegram, github: fakeGithub(), now: fixedNow, logger: silentLogger }).poll();

  const payload = telegram.state.sent[0];
  assert.equal(payload.chat_id, -1001234567890);
  assert.equal(payload.reply_parameters.message_id, 60);
  assert.equal(payload.message_thread_id, 777);
  assert.deepEqual(payload.link_preview_options, { is_disabled: true });
  assert.equal(Object.hasOwn(payload, 'parse_mode'), false);
});

test('welcome service updates receive only the approved generic outreach reply', async () => {
  const telegram = fakeTelegram({
    updates: [update(70, official('', 70, {
      new_chat_members: [{ id: 777888999, first_name: 'Sensitive Name' }],
    }))],
  });

  await createWorker({ telegram, github: fakeGithub(), now: fixedNow, logger: silentLogger }).poll();

  assert.equal(telegram.state.sent.length, 1);
  assert.match(telegram.state.sent[0].text, /SUDHARMA NETWORK — PUBLIC TESTNET/);
  assert.match(telegram.state.sent[0].text, /https:\/\/feature-website-foundation\.d2mqyt0bt8sl9s\.amplifyapp\.com\//);
  assert.doesNotMatch(telegram.state.sent[0].text, /777888999|Sensitive Name/);
});

test('invalid report receives public-GitHub and sensitive-data guidance without creating an issue', async () => {
  const telegram = fakeTelegram({ updates: [update(80, official('/report too short', 80))] });
  const github = fakeGithub();

  await createWorker({ telegram, github, now: fixedNow, logger: silentLogger }).poll();

  assert.equal(telegram.state.sent.length, 1);
  assert.equal(github.state.created.length, 0);
  assert.match(telegram.state.sent[0].text, /public GitHub/i);
  assert.match(telegram.state.sent[0].text, /private key|seed phrase|token/i);
  assert.deepEqual(telegram.state.getUpdatesCalls.at(-1), { offset: 81, limit: 1, timeout: 0 });
});

test('twenty-reply cap stops before the twenty-first actionable update and leaves it unacknowledged', async () => {
  const updates = Array.from({ length: 21 }, (_, index) => {
    const id = index + 1;
    return update(id, official('/help', id));
  });
  const telegram = fakeTelegram({ updates });

  await createWorker({ telegram, github: fakeGithub(), now: fixedNow, logger: silentLogger }).poll();

  assert.equal(telegram.state.sent.length, 20);
  assert.deepEqual(telegram.state.getUpdatesCalls.at(-1), { offset: 21, limit: 1, timeout: 0 });
});

test('informational command after twenty successful replies is not silently acknowledged', async () => {
  const updates = Array.from({ length: 20 }, (_, index) => update(index + 1, official('/help', index + 1)));
  updates.push(update(25, official('/miner', 25)));
  updates.push(update(30, otherGroup('/help', 30)));
  const telegram = fakeTelegram({ updates });

  await createWorker({ telegram, github: fakeGithub(), now: fixedNow, logger: silentLogger }).poll();

  assert.equal(telegram.state.sent.length, 20);
  assert.deepEqual(telegram.state.getUpdatesCalls.at(-1), { offset: 21, limit: 1, timeout: 0 });
});

test('/report creates one structured GitHub issue and replies with its URL', async () => {
  const telegram = fakeTelegram({
    updates: [update(100, official('/report miner exits after startup on RX 580 with OpenCL initialization failure', 100))],
  });
  const github = fakeGithub({ createdUrl: 'https://github.com/sudharma-networks/sudharma/issues/999' });

  await createWorker({ telegram, github, now: fixedNow, logger: silentLogger }).poll();

  assert.equal(github.state.created.length, 1);
  assert.match(github.state.created[0].title, /^Telegram report: /);
  assert.match(github.state.created[0].body, /update_id=100/);
  assert.match(github.state.created[0].body, /https:\/\/t\.me\/sudharma_community\/100/);
  assert.equal(telegram.state.sent.length, 1);
  assert.match(telegram.state.sent[0].text, /https:\/\/github\.com\/sudharma-networks\/sudharma\/issues\/999/);
  assert.deepEqual(telegram.state.getUpdatesCalls.at(-1), { offset: 101, limit: 1, timeout: 0 });
});

test('/report retry reuses an existing issue with the same update marker', async () => {
  const existingUrl = 'https://github.com/sudharma-networks/sudharma/issues/777';
  const github = fakeGithub({
    recentIssues: [{
      html_url: existingUrl,
      body: `${reportMarker(110)}\nexisting report`,
      created_at: '2026-08-30T01:45:00Z',
    }],
  });
  const telegram = fakeTelegram({
    updates: [update(110, official('/report retry-safe miner failure with enough detail for intake', 110))],
  });

  await createWorker({ telegram, github, now: fixedNow, logger: silentLogger }).poll();

  assert.equal(github.state.created.length, 0);
  assert.equal(telegram.state.sent.length, 1);
  assert.match(telegram.state.sent[0].text, /issues\/777/);
  assert.deepEqual(telegram.state.getUpdatesCalls.at(-1), { offset: 111, limit: 1, timeout: 0 });
});

test('rolling one-hour limit throttles the eleventh Telegram-created report', async () => {
  const recentIssues = Array.from({ length: 10 }, (_, index) => ({
    html_url: `https://github.com/sudharma-networks/sudharma/issues/${700 + index}`,
    body: `${reportMarker(200 + index)}\nexisting report`,
    created_at: `2026-08-30T01:${String(40 + index).padStart(2, '0')}:00Z`,
  }));
  const github = fakeGithub({ recentIssues });
  const telegram = fakeTelegram({
    updates: [update(120, official('/report this valid report should be throttled by the rolling hour limit', 120))],
  });

  await createWorker({ telegram, github, now: fixedNow, logger: silentLogger }).poll();

  assert.equal(github.state.created.length, 0);
  assert.equal(telegram.state.sent.length, 1);
  assert.match(telegram.state.sent[0].text, /rate-limited/i);
  assert.deepEqual(telegram.state.getUpdatesCalls.at(-1), { offset: 121, limit: 1, timeout: 0 });
});

test('per-run report limit creates at most three issues and throttles the fourth', async () => {
  const telegram = fakeTelegram({
    updates: [1, 2, 3, 4].map((id) => update(130 + id, official(`/report valid report number ${id} with enough detail for controlled intake`, 130 + id))),
  });
  const github = fakeGithub();

  await createWorker({ telegram, github, now: fixedNow, logger: silentLogger }).poll();

  assert.equal(github.state.created.length, 3);
  assert.equal(telegram.state.sent.length, 4);
  assert.match(telegram.state.sent[3].text, /rate-limited/i);
  assert.deepEqual(telegram.state.getUpdatesCalls.at(-1), { offset: 135, limit: 1, timeout: 0 });
});

test('GitHub issue creation failure leaves the report update unacknowledged for retry', async () => {
  const telegram = fakeTelegram({
    updates: [update(140, official('/report GitHub failure path must leave this report available for retry', 140))],
  });
  const github = fakeGithub({ failCreate: true });

  await createWorker({ telegram, github, now: fixedNow, logger: silentLogger }).poll();

  assert.equal(github.state.created.length, 1);
  assert.equal(telegram.state.sent.length, 0);
  assert.deepEqual(telegram.state.getUpdatesCalls, [{ limit: 100, timeout: 0 }]);
});

test('report is not created when reply budget is already exhausted', async () => {
  const updates = Array.from({ length: 20 }, (_, index) => update(index + 1, official('/help', index + 1)));
  updates.push(update(150, official('/report this report must wait because reply capacity is exhausted', 150)));
  const telegram = fakeTelegram({ updates });
  const github = fakeGithub();

  await createWorker({ telegram, github, now: fixedNow, logger: silentLogger }).poll();

  assert.equal(telegram.state.sent.length, 20);
  assert.equal(github.state.created.length, 0);
  assert.equal(github.state.listCalls.length, 0);
  assert.deepEqual(telegram.state.getUpdatesCalls.at(-1), { offset: 21, limit: 1, timeout: 0 });
});

test('Telegram adapter never includes token in thrown API errors', async () => {
  const token = '123456:SECRET_VALUE_THAT_MUST_NOT_LEAK';
  const adapter = createTelegramAdapter({
    token,
    fetchImpl: async () => ({
      ok: true,
      async json() {
        return { ok: false, description: `bad token ${token}` };
      },
    }),
  });

  await assert.rejects(
    () => adapter.getMe(),
    (error) => {
      assert.doesNotMatch(String(error), /SECRET_VALUE_THAT_MUST_NOT_LEAK/);
      assert.match(String(error), /Telegram API getMe failed/);
      return true;
    },
  );
});

test('Telegram adapter sends JSON Bot API requests and returns result objects', async () => {
  const calls = [];
  const adapter = createTelegramAdapter({
    token: 'safe-test-token',
    fetchImpl: async (url, options) => {
      calls.push({ url, options });
      return {
        ok: true,
        async json() {
          return { ok: true, result: { id: 1, username: 'SudharmaNetworkBot' } };
        },
      };
    },
  });

  const result = await adapter.getMe();
  assert.equal(result.username, 'SudharmaNetworkBot');
  assert.equal(calls.length, 1);
  assert.match(calls[0].url, /\/getMe$/);
  assert.equal(calls[0].options.method, 'POST');
  assert.equal(calls[0].options.headers['Content-Type'], 'application/json');
});

test('GitHub adapter creates public issues with repository-scoped API path', async () => {
  const calls = [];
  const adapter = createGithubAdapter({
    token: 'github-test-token',
    repository: 'sudharma-networks/sudharma',
    fetchImpl: async (url, options) => {
      calls.push({ url, options });
      return {
        ok: true,
        status: 201,
        async json() {
          return { html_url: 'https://github.com/sudharma-networks/sudharma/issues/901', title: 't', body: 'b', created_at: '2026-08-30T02:00:00Z' };
        },
      };
    },
  });

  const created = await adapter.createIssue({ title: 'Telegram report: test', body: 'body' });
  assert.match(created.html_url, /issues\/901/);
  assert.equal(calls.length, 1);
  assert.match(calls[0].url, /repos\/sudharma-networks\/sudharma\/issues$/);
  assert.equal(calls[0].options.method, 'POST');
  assert.equal(JSON.parse(calls[0].options.body).title, 'Telegram report: test');
});

test('GitHub adapter paginates recent issues and excludes pull requests', async () => {
  const calls = [];
  const firstPage = Array.from({ length: 100 }, (_, index) => ({
    html_url: `https://github.com/sudharma-networks/sudharma/issues/${index + 1}`,
    body: 'ordinary issue',
    created_at: '2026-08-30T01:30:00Z',
  }));
  firstPage[0] = { ...firstPage[0], pull_request: { url: 'https://api.github.com/pulls/1' } };

  const adapter = createGithubAdapter({
    token: 'github-test-token',
    repository: 'sudharma-networks/sudharma',
    fetchImpl: async (url) => {
      calls.push(url);
      const page = new URL(url).searchParams.get('page');
      return {
        ok: true,
        status: 200,
        async json() {
          return page === '1'
            ? firstPage
            : [{ html_url: 'https://github.com/sudharma-networks/sudharma/issues/101', body: 'last', created_at: '2026-08-30T01:20:00Z' }];
        },
      };
    },
  });

  const issues = await adapter.listRecentIssues({ since: '2026-08-30T01:00:00Z' });
  assert.equal(calls.length, 2);
  assert.equal(issues.length, 100);
  assert.equal(issues.some((issue) => issue.pull_request), false);
});

test('GitHub adapter error never includes the GitHub token', async () => {
  const token = 'ghs_SECRET_VALUE_THAT_MUST_NOT_LEAK';
  const adapter = createGithubAdapter({
    token,
    repository: 'sudharma-networks/sudharma',
    fetchImpl: async () => ({
      ok: false,
      status: 500,
      async json() {
        return { message: token };
      },
    }),
  });

  await assert.rejects(
    () => adapter.createIssue({ title: 'x', body: 'y' }),
    (error) => {
      assert.doesNotMatch(String(error), /SECRET_VALUE_THAT_MUST_NOT_LEAK/);
      assert.match(String(error), /GitHub API/i);
      return true;
    },
  );
});
