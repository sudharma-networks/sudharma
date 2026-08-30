'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const { createWorker } = require('./community-worker.js');

function fakeTelegram() {
  const state = { sent: [], getUpdatesCalls: [] };
  return {
    state,
    async getMe() {
      return { id: 12345, username: 'SudharmaNetwork2026Bot' };
    },
    async getUpdates(args) {
      state.getUpdatesCalls.push({ ...args });
      if (Object.hasOwn(args, 'offset')) return [];
      return [{
        update_id: 689350004,
        message: {
          message_id: 44,
          date: 1788048000,
          chat: {
            id: -1001234567890,
            type: 'supergroup',
            username: 'sudharma_community',
          },
          text: '/report too short',
        },
      }];
    },
    async sendMessage(payload) {
      state.sent.push(structuredClone(payload));
      return { message_id: 9001 };
    },
  };
}

const fakeGithub = {
  async listRecentIssues() { return []; },
  async createIssue() { throw new Error('must not create an issue for invalid report'); },
};

test('community replies allow delivery when the original reply target is no longer available', async () => {
  const telegram = fakeTelegram();
  const result = await createWorker({
    telegram,
    github: fakeGithub,
    now: () => new Date('2026-08-30T03:30:00Z'),
    logger: { info() {}, warn() {}, error() {} },
  }).poll();

  assert.equal(result.replies, 1);
  assert.equal(telegram.state.sent.length, 1);
  assert.deepEqual(telegram.state.sent[0].reply_parameters, {
    message_id: 44,
    allow_sending_without_reply: true,
  });
  assert.deepEqual(telegram.state.getUpdatesCalls.at(-1), {
    offset: 689350005,
    limit: 1,
    timeout: 0,
  });
});
