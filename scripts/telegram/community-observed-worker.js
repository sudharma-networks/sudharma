'use strict';

const {
  createTelegramAdapter,
  createGithubAdapter,
  createWorker,
  readRuntimeConfig,
} = require('./community-worker.js');

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
          logger.error(`Telegram community boundary failed: ${scope}.${operation}`);
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

  const telegram = instrumentDependency(
    'telegram',
    rawTelegram,
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
  instrumentDependency,
};
