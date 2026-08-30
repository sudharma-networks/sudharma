'use strict';

const {
  createTelegramAdapter,
  createGithubAdapter,
  createWorker,
  readRuntimeConfig,
} = require('./community-worker.js');

const SAFE_DIAGNOSTIC_CODE_RE = /^telegram-api-[1-5]\d{2}-(?:reply-target-missing|thread-missing|chat-not-found|membership|permission-denied|rate-limited|other)$/;

function withReplyTargetFallback(telegram) {
  if (!telegram || typeof telegram !== 'object' || typeof telegram.sendMessage !== 'function') {
    throw new Error('Telegram reply-fallback dependency is invalid');
  }

  const wrapped = Object.create(telegram);
  wrapped.sendMessage = async (payload) => {
    if (!payload || typeof payload !== 'object') {
      return telegram.sendMessage(payload);
    }

    const nextPayload = { ...payload };
    if (payload.reply_parameters && typeof payload.reply_parameters === 'object') {
      nextPayload.reply_parameters = {
        ...payload.reply_parameters,
        allow_sending_without_reply: true,
      };
    }
    return telegram.sendMessage(nextPayload);
  };
  return wrapped;
}

function instrumentDependency(scope, dependency, operations, logger = console) {
  if (!dependency || typeof dependency !== 'object') {
    throw new Error('Observed dependency is invalid');
  }
  if (!Array.isArray(operations)) {
    throw new Error('Observed operation list is invalid');
  }

  const wrapped = Object.create(dependency);
  for (const operation of operations) {
    if (typeof dependency[operation] !== 'function') continue;
    wrapped[operation] = async (...args) => {
      try {
        const result = await dependency[operation](...args);
        if (logger && typeof logger.info === 'function') {
          logger.info(`Telegram community boundary ok: ${scope}.${operation}`);
        }
        return result;
      } catch (error) {
        if (logger && typeof logger.error === 'function') {
          const diagnosticCode = typeof error?.diagnosticCode === 'string' && SAFE_DIAGNOSTIC_CODE_RE.test(error.diagnosticCode)
            ? ` code=${error.diagnosticCode}`
            : '';
          logger.error(`Telegram community boundary failed: ${scope}.${operation}${diagnosticCode}`);
        }
        throw error;
      }
    };
  }
  return wrapped;
}

async function main() {
  const config = readRuntimeConfig(process.env);
  const rawTelegram = createTelegramAdapter({ token: config.telegramToken });
  const rawGithub = createGithubAdapter({ token: config.githubToken, repository: config.repository });

  const resilientTelegram = withReplyTargetFallback(rawTelegram);
  const telegram = instrumentDependency(
    'telegram',
    resilientTelegram,
    ['getMe', 'getWebhookInfo', 'deleteWebhook', 'getUpdates', 'sendMessage'],
  );
  const github = instrumentDependency(
    'github',
    rawGithub,
    ['listRecentIssues', 'listRecentAutomationIssues', 'createIssue'],
  );

  const worker = createWorker({ telegram, github });
  if (config.mode === 'bootstrap') {
    const result = await worker.bootstrap();
    console.log(`Telegram community bootstrap completed for @${result.botUsername}`);
    return;
  }

  const result = await worker.poll();
  console.log(`Telegram community poll completed: replies=${result.replies}, reports=${result.createdReports}, last_confirmed=${result.lastConfirmed ?? 'none'}`);
}

if (require.main === module) {
  main().catch((error) => {
    console.error(`Telegram community observed worker failed: ${error.message}`);
    process.exitCode = 1;
  });
}

module.exports = {
  withReplyTargetFallback,
  instrumentDependency,
};
