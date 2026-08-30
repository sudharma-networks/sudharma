'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const { createTelegramAdapter } = require('./community-worker.js');

test('Telegram adapter classifies a missing reply target without exposing raw API description', async () => {
  const secret = 'RAW_DESCRIPTION_MUST_NOT_LEAK';
  const adapter = createTelegramAdapter({
    token: 'safe-test-token',
    fetchImpl: async () => ({
      ok: true,
      status: 200,
      async json() {
        return {
          ok: false,
          error_code: 400,
          description: `Bad Request: message to be replied not found ${secret}`,
        };
      },
    }),
  });

  await assert.rejects(
    () => adapter.sendMessage({ chat_id: -1001, text: 'safe', reply_parameters: { message_id: 99 } }),
    (error) => {
      assert.equal(error.diagnosticCode, 'telegram-api-400-reply-target-missing');
      assert.doesNotMatch(String(error), new RegExp(secret));
      return true;
    },
  );
});

test('Telegram adapter uses a generic safe code for an unrecognized Telegram API failure', async () => {
  const secret = 'UNKNOWN_RAW_DESCRIPTION_MUST_NOT_LEAK';
  const adapter = createTelegramAdapter({
    token: 'safe-test-token',
    fetchImpl: async () => ({
      ok: false,
      status: 429,
      async json() {
        return {
          ok: false,
          error_code: 429,
          description: `Too Many Requests: ${secret}`,
        };
      },
    }),
  });

  await assert.rejects(
    () => adapter.sendMessage({ chat_id: -1001, text: 'safe' }),
    (error) => {
      assert.equal(error.diagnosticCode, 'telegram-api-429-other');
      assert.doesNotMatch(String(error), new RegExp(secret));
      return true;
    },
  );
});
