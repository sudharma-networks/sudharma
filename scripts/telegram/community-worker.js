'use strict';

const {
  parseCommunityUpdate,
  validateReportText,
  staticReply,
} = require('./community-core.js');

const MAX_REPLIES_PER_RUN = 20;

function createTelegramAdapter({ token, fetchImpl = globalThis.fetch }) {
  if (typeof token !== 'string' || token.length === 0) {
    throw new Error('Telegram bot token is required');
  }
  if (typeof fetchImpl !== 'function') {
    throw new Error('A fetch implementation is required');
  }

  async function call(method, payload = {}) {
    let response;
    try {
      response = await fetchImpl(`https://api.telegram.org/bot${token}/${method}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
    } catch {
      throw new Error(`Telegram API ${method} failed`);
    }

    let data;
    try {
      data = await response.json();
    } catch {
      throw new Error(`Telegram API ${method} failed`);
    }

    if (!response.ok || !data || data.ok !== true) {
      throw new Error(`Telegram API ${method} failed`);
    }
    return data.result;
  }

  return {
    getMe() {
      return call('getMe');
    },
    getWebhookInfo() {
      return call('getWebhookInfo');
    },
    deleteWebhook({ dropPendingUpdates }) {
      return call('deleteWebhook', { drop_pending_updates: Boolean(dropPendingUpdates) });
    },
    getUpdates({ offset, limit, timeout } = {}) {
      const payload = {};
      if (offset !== undefined) payload.offset = offset;
      if (limit !== undefined) payload.limit = limit;
      if (timeout !== undefined) payload.timeout = timeout;
      return call('getUpdates', payload);
    },
    sendMessage(payload) {
      return call('sendMessage', payload);
    },
  };
}

function replyPayload(message, text) {
  const payload = {
    chat_id: message.chat.id,
    text,
    reply_parameters: { message_id: message.message_id },
    link_preview_options: { is_disabled: true },
  };
  if (Number.isInteger(message.message_thread_id)) {
    payload.message_thread_id = message.message_thread_id;
  }
  return payload;
}

function createWorker({ telegram, github, now = () => new Date(), logger = console }) {
  if (!telegram || typeof telegram.getMe !== 'function' || typeof telegram.getUpdates !== 'function' || typeof telegram.sendMessage !== 'function') {
    throw new Error('Telegram dependency is invalid');
  }
  if (!github || typeof github !== 'object') {
    throw new Error('GitHub dependency is invalid');
  }
  if (typeof now !== 'function') {
    throw new Error('now must be a function');
  }

  async function poll() {
    const bot = await telegram.getMe();
    if (!bot || typeof bot.username !== 'string' || bot.username.length === 0) {
      throw new Error('Telegram bot username is unavailable');
    }

    const fetched = await telegram.getUpdates({ limit: 100, timeout: 0 });
    if (!Array.isArray(fetched)) {
      throw new Error('Telegram getUpdates result must be an array');
    }

    const updates = [...fetched].sort((left, right) => left.update_id - right.update_id);
    let replies = 0;
    let lastConfirmed = null;

    async function sendReply(message, text) {
      if (replies >= MAX_REPLIES_PER_RUN) {
        throw new Error('Telegram community reply limit reached');
      }
      await telegram.sendMessage(replyPayload(message, text));
      replies += 1;
    }

    async function handleUpdate(update) {
      const action = parseCommunityUpdate(update, bot.username);
      if (action.kind === 'ignore') return;

      if (action.kind === 'welcome') {
        await sendReply(action.message, staticReply('welcome'));
        return;
      }

      if (action.kind !== 'command') return;

      if (action.command === 'report') {
        try {
          validateReportText(action.args);
        } catch {
          await sendReply(action.message, staticReply('report-usage'));
          return;
        }
        throw new Error('Telegram community report processing is not implemented yet');
      }

      await sendReply(action.message, staticReply(action.command));
    }

    for (const update of updates) {
      if (!update || !Number.isInteger(update.update_id) || update.update_id < 0) {
        if (logger && typeof logger.error === 'function') {
          logger.error('Telegram community update has an invalid update_id');
        }
        break;
      }
      try {
        await handleUpdate(update);
        lastConfirmed = update.update_id;
      } catch {
        if (logger && typeof logger.error === 'function') {
          logger.error('Telegram community update processing failed', { updateId: update.update_id });
        }
        break;
      }
    }

    if (lastConfirmed !== null) {
      await telegram.getUpdates({ offset: lastConfirmed + 1, limit: 1, timeout: 0 });
    }

    return { lastConfirmed, replies };
  }

  return { poll };
}

module.exports = {
  MAX_REPLIES_PER_RUN,
  createTelegramAdapter,
  createWorker,
};
