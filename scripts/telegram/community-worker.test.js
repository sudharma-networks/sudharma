'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const {
  createWorker,
  createTelegramAdapter,
} = require('./community-worker.js');

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

function fakeGithub() {
  return {
    async listRecentIssues() {
      return [];
    },
    async createIssue() {
      throw new Error('report creation is not expected in Task 2 tests');
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

test('welcome service updates receive only the generic welcome reply', async () => {
  const telegram = fakeTelegram({
    updates: [update(70, official('', 70, {
      new_chat_members: [{ id: 777888999, first_name: 'Sensitive Name' }],
    }))],
  });

  await createWorker({ telegram, github: fakeGithub(), now: fixedNow, logger: silentLogger }).poll();

  assert.equal(telegram.state.sent.length, 1);
  assert.match(telegram.state.sent[0].text, /welcome/i);
  assert.doesNotMatch(telegram.state.sent[0].text, /777888999|Sensitive Name/);
});

test('invalid report receives public-GitHub and sensitive-data guidance without creating an issue', async () => {
  const telegram = fakeTelegram({ updates: [update(80, official('/report too short', 80))] });
  const github = fakeGithub();

  await createWorker({ telegram, github, now: fixedNow, logger: silentLogger }).poll();

  assert.equal(telegram.state.sent.length, 1);
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
