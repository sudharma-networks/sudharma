'use strict';

const fs = require('node:fs');
const { parseTelegramMessage, isTrustedAssociation } = require('./bridge-core.js');

const LABEL_BY_MODE = Object.freeze({
  'dry-run': 'telegram:dry-run',
  publish: 'telegram:publish-approved',
});

function fail(message) {
  process.stderr.write(`Telegram bridge validation failed: ${message}\n`);
  process.exit(1);
}

function main() {
  const eventPath = process.env.GITHUB_EVENT_PATH;
  const mode = process.env.TELEGRAM_MODE;
  const messageFile = process.env.TELEGRAM_MESSAGE_FILE;

  if (!eventPath) fail('GITHUB_EVENT_PATH is required');
  if (!Object.hasOwn(LABEL_BY_MODE, mode)) fail('TELEGRAM_MODE must be dry-run or publish');
  if (!messageFile) fail('TELEGRAM_MESSAGE_FILE is required');

  let event;
  try {
    event = JSON.parse(fs.readFileSync(eventPath, 'utf8'));
  } catch {
    fail('GitHub event JSON could not be read');
  }

  if (event?.action !== 'labeled' || !event?.issue || typeof event.issue !== 'object') {
    fail('expected a labeled GitHub issue event');
  }
  if (event.issue.pull_request) {
    fail('pull request issue payloads are not allowed');
  }
  if (!isTrustedAssociation(event.issue.author_association)) {
    fail('issue author association is not trusted');
  }

  const expectedLabel = LABEL_BY_MODE[mode];
  if (event?.label?.name !== expectedLabel) {
    fail(`event label must be exactly ${expectedLabel}`);
  }

  let message;
  try {
    message = parseTelegramMessage(event.issue.body);
  } catch (error) {
    fail(error instanceof Error ? error.message : 'Telegram message is invalid');
  }

  fs.writeFileSync(messageFile, message, { encoding: 'utf8', mode: 0o600 });
  process.stdout.write(`Validated Telegram ${mode} request (${[...message].length} Unicode code points).\n`);
}

main();
