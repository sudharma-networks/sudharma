'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const {
  createWorker,
  readRuntimeConfig,
} = require('./community-worker.js');

function fakeTelegram({ webhookUrl = '', botUsername = 'SudharmaNetworkBot' } = {}) {
  const state = {
    getMeCalls: 0,
    webhookInfoCalls: 0,
    deletedWebhook: [],
    getUpdatesCalls: [],
    sent: [],
  };
  return {
    state,
    async getMe() {
      state.getMeCalls += 1;
      return { id: 123, username: botUsername };
    },
    async getWebhookInfo() {
      state.webhookInfoCalls += 1;
      return { url: webhookUrl };
    },
    async deleteWebhook(args) {
      state.deletedWebhook.push(structuredClone(args));
      return true;
    },
    async getUpdates(args) {
      state.getUpdatesCalls.push(structuredClone(args));
      return [];
    },
    async sendMessage(payload) {
      state.sent.push(structuredClone(payload));
      return { message_id: 1 };
    },
  };
}

function fakeGithub() {
  const state = { listCalls: 0, createCalls: 0 };
  return {
    state,
    async listRecentIssues() {
      state.listCalls += 1;
      return [];
    },
    async createIssue() {
      state.createCalls += 1;
      return { html_url: 'https://example.invalid/issue' };
    },
  };
}

const fixedNow = () => new Date('2026-08-30T02:00:00Z');
const silentLogger = { info() {}, warn() {}, error() {} };

test('bootstrap fails closed when an existing webhook is configured', async () => {
  const telegram = fakeTelegram({ webhookUrl: 'https://existing.example/hook' });
  const github = fakeGithub();
  const worker = createWorker({ telegram, github, now: fixedNow, logger: silentLogger });

  await assert.rejects(() => worker.bootstrap(), /webhook/i);

  assert.equal(telegram.state.getMeCalls, 1);
  assert.equal(telegram.state.webhookInfoCalls, 1);
  assert.deepEqual(telegram.state.deletedWebhook, []);
  assert.deepEqual(telegram.state.getUpdatesCalls, []);
  assert.deepEqual(telegram.state.sent, []);
  assert.equal(github.state.listCalls, 0);
  assert.equal(github.state.createCalls, 0);
});

test('bootstrap drops pre-activation pending updates only when no webhook exists', async () => {
  const telegram = fakeTelegram({ webhookUrl: '' });
  const github = fakeGithub();
  const worker = createWorker({ telegram, github, now: fixedNow, logger: silentLogger });

  const result = await worker.bootstrap();

  assert.equal(result.botUsername, 'SudharmaNetworkBot');
  assert.equal(telegram.state.getMeCalls, 1);
  assert.equal(telegram.state.webhookInfoCalls, 1);
  assert.deepEqual(telegram.state.deletedWebhook, [{ dropPendingUpdates: true }]);
  assert.deepEqual(telegram.state.getUpdatesCalls, []);
  assert.deepEqual(telegram.state.sent, []);
  assert.equal(github.state.listCalls, 0);
  assert.equal(github.state.createCalls, 0);
});

test('bootstrap rejects missing bot username before touching webhook state', async () => {
  const telegram = fakeTelegram({ webhookUrl: '', botUsername: '' });
  const worker = createWorker({ telegram, github: fakeGithub(), now: fixedNow, logger: silentLogger });

  await assert.rejects(() => worker.bootstrap(), /username/i);

  assert.equal(telegram.state.getMeCalls, 1);
  assert.equal(telegram.state.webhookInfoCalls, 0);
  assert.deepEqual(telegram.state.deletedWebhook, []);
});

test('runtime config accepts only bootstrap or poll with every required variable present', () => {
  const env = {
    TELEGRAM_BOT_TOKEN: 'test-token',
    GITHUB_TOKEN: 'github-token',
    GITHUB_REPOSITORY: 'sudharma-networks/sudharma',
    COMMUNITY_MODE: 'bootstrap',
  };

  assert.deepEqual(readRuntimeConfig(env), {
    telegramToken: 'test-token',
    githubToken: 'github-token',
    repository: 'sudharma-networks/sudharma',
    mode: 'bootstrap',
  });
  assert.equal(readRuntimeConfig({ ...env, COMMUNITY_MODE: 'poll' }).mode, 'poll');
  assert.throws(() => readRuntimeConfig({ ...env, COMMUNITY_MODE: 'delete-everything' }), /COMMUNITY_MODE/i);
});

test('runtime config rejects each missing required variable without revealing any secret value', () => {
  const complete = {
    TELEGRAM_BOT_TOKEN: 'tg_SECRET_VALUE',
    GITHUB_TOKEN: 'gh_SECRET_VALUE',
    GITHUB_REPOSITORY: 'sudharma-networks/sudharma',
    COMMUNITY_MODE: 'poll',
  };

  for (const key of Object.keys(complete)) {
    const env = { ...complete };
    delete env[key];
    assert.throws(
      () => readRuntimeConfig(env),
      (error) => {
        assert.match(String(error), new RegExp(key));
        assert.doesNotMatch(String(error), /tg_SECRET_VALUE|gh_SECRET_VALUE/);
        return true;
      },
    );
  }
});
