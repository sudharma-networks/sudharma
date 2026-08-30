'use strict';

const OFFICIAL_GROUP_USERNAME = 'sudharma_community';
const REPORT_MIN_CODE_POINTS = 20;
const REPORT_MAX_CODE_POINTS = 2000;
const REPORT_MARKER_PREFIX = '<!-- sudharma-telegram-community-report:v1 update_id=';
const SUPPORTED_COMMANDS = new Set(['help', 'rules', 'testnet', 'miner', 'report']);

function codePointLength(value) {
  return [...String(value)].length;
}

function isOfficialChat(message) {
  if (!message || typeof message !== 'object' || !message.chat || typeof message.chat !== 'object') {
    return false;
  }
  const type = message.chat.type;
  const username = typeof message.chat.username === 'string' ? message.chat.username.toLowerCase() : '';
  return (type === 'group' || type === 'supergroup') && username === OFFICIAL_GROUP_USERNAME;
}

function normalizeBotUsername(value) {
  if (typeof value !== 'string') return '';
  return value.replace(/^@/, '').toLowerCase();
}

function parseCommunityUpdate(update, botUsername) {
  const message = update && typeof update === 'object' ? update.message : null;
  if (!isOfficialChat(message)) return { kind: 'ignore' };

  if (Array.isArray(message.new_chat_members) && message.new_chat_members.length > 0) {
    return { kind: 'welcome', message };
  }

  const source = typeof message.text === 'string'
    ? message.text
    : (typeof message.caption === 'string' ? message.caption : '');
  const text = source.trimStart();
  if (!text.startsWith('/')) return { kind: 'ignore' };

  const match = text.match(/^\/([A-Za-z0-9_]+)(?:@([A-Za-z0-9_]+))?(?:\s+([\s\S]*))?$/);
  if (!match) return { kind: 'ignore' };

  const rawCommand = match[1].toLowerCase();
  const target = match[2] ? match[2].toLowerCase() : '';
  const ownUsername = normalizeBotUsername(botUsername);
  if (target && (!ownUsername || target !== ownUsername)) {
    return { kind: 'ignore' };
  }

  const command = SUPPORTED_COMMANDS.has(rawCommand) ? rawCommand : 'help';
  return {
    kind: 'command',
    command,
    args: (match[3] || '').trim(),
    message,
  };
}

function validateReportText(value) {
  const text = typeof value === 'string' ? value.trim() : '';
  const length = codePointLength(text);
  if (length < REPORT_MIN_CODE_POINTS) {
    throw new Error(`Report must contain at least ${REPORT_MIN_CODE_POINTS} Unicode code points`);
  }
  if (length > REPORT_MAX_CODE_POINTS) {
    throw new Error(`Report must contain no more than ${REPORT_MAX_CODE_POINTS} Unicode code points`);
  }
  return text;
}

function neutralizeGithubMentions(value) {
  return String(value).replace(/@/g, '@\u200b');
}

function telegramMessageLink(message) {
  if (!isOfficialChat(message)) {
    throw new Error('Telegram message is not from the official community group');
  }
  if (!Number.isInteger(message.message_id) || message.message_id <= 0) {
    throw new Error('Telegram message_id is invalid');
  }
  return `https://t.me/${OFFICIAL_GROUP_USERNAME}/${message.message_id}`;
}

function reportMarker(updateId) {
  if (!Number.isInteger(updateId) || updateId < 0) {
    throw new Error('Telegram update_id must be a non-negative integer');
  }
  return `${REPORT_MARKER_PREFIX}${updateId} -->`;
}

function hasReportMarker(issueBody, updateId) {
  return typeof issueBody === 'string' && issueBody.includes(reportMarker(updateId));
}

function shortenCodePoints(value, maximum) {
  return [...value].slice(0, maximum).join('');
}

function quoteMarkdown(value) {
  return value.split('\n').map((line) => `> ${line}`).join('\n');
}

function buildReportIssue({ updateId, reportText, message, now }) {
  const validated = validateReportText(reportText);
  const safeText = neutralizeGithubMentions(validated);
  const sourceLink = telegramMessageLink(message);
  if (!Number.isFinite(message.date)) {
    throw new Error('Telegram message date is invalid');
  }

  const telegramTime = new Date(message.date * 1000);
  if (Number.isNaN(telegramTime.getTime())) {
    throw new Error('Telegram message date is invalid');
  }
  const ingestionTime = now instanceof Date ? now : new Date(now);
  if (Number.isNaN(ingestionTime.getTime())) {
    throw new Error('Ingestion time is invalid');
  }

  const normalizedTitle = safeText.replace(/\s+/g, ' ').trim();
  const title = `Telegram report: ${shortenCodePoints(normalizedTitle, 80)}`;
  const body = [
    reportMarker(updateId),
    '',
    '## Community report',
    '',
    quoteMarkdown(safeText),
    '',
    `Source Telegram message: ${sourceLink}`,
    `Telegram message time (UTC): ${telegramTime.toISOString()}`,
    `Ingested at (UTC): ${ingestionTime.toISOString()}`,
    '',
    'Attachments remain in Telegram and are not copied into GitHub by Phase 2A.',
    '',
    'Context: Sudharma Network is pre-mainnet/public-testnet experimental software.',
  ].join('\n');

  return { title, body };
}

const REPLIES = Object.freeze({
  help: [
    'Sudharma Network community bot — Phase 2A.',
    '/help — commands and safety notice',
    '/rules — community rules',
    '/testnet — public-testnet information',
    '/miner — experimental test-mining resources',
    '/report <problem> — create a public GitHub issue',
    '',
    'Sudharma is pre-mainnet/public-testnet experimental software. Never share seed phrases, private keys, passwords, tokens, or personal information.',
  ].join('\n'),
  rules: [
    'Sudharma community rules:',
    '• Be respectful and stay on topic.',
    '• No spam, scams, impersonation, or misleading financial claims.',
    '• Never share seed phrases, private keys, passwords, tokens, or personal information.',
    '• Sudharma is pre-mainnet/public-testnet experimental software.',
  ].join('\n'),
  testnet: [
    'Sudharma Network is currently pre-mainnet/public-testnet experimental software.',
    'Source code and project information: https://github.com/sudharma-networks/sudharma',
    'Use /report <problem> to send a non-sensitive testnet problem to the public GitHub issue tracker.',
  ].join('\n'),
  miner: [
    'Sudharma test mining is experimental.',
    'Releases: https://github.com/sudharma-networks/sudharma/releases',
    'Use /report <problem> for reproducible miner/testnet problems. No earnings claims are made.',
  ].join('\n'),
  welcome: [
    'Welcome to the Sudharma Network community.',
    'Sudharma is a pre-mainnet/public-testnet experimental project.',
    'Use /help to see the commands available from this bot.',
  ].join('\n'),
  'report-usage': [
    'Usage: /report <problem details> using 20–2000 characters.',
    'Accepted report text is copied into the public GitHub repository.',
    'Do not include seed phrases, private keys, passwords, tokens, personal information, or other sensitive data.',
  ].join('\n'),
  throttle: [
    'Automated Telegram report intake is temporarily rate-limited.',
    'Please report manually at https://github.com/sudharma-networks/sudharma/issues and do not include secrets or personal information.',
  ].join('\n'),
});

function staticReply(kind) {
  if (!Object.hasOwn(REPLIES, kind)) {
    throw new Error(`Unknown community reply kind: ${kind}`);
  }
  return REPLIES[kind];
}

function canCreateReport({ createdThisRun, createdLastHour }) {
  if (createdThisRun >= 3) return { allowed: false, reason: 'run-limit' };
  if (createdLastHour >= 10) return { allowed: false, reason: 'hour-limit' };
  return { allowed: true };
}

module.exports = {
  OFFICIAL_GROUP_USERNAME,
  REPORT_MIN_CODE_POINTS,
  REPORT_MAX_CODE_POINTS,
  isOfficialChat,
  parseCommunityUpdate,
  validateReportText,
  neutralizeGithubMentions,
  telegramMessageLink,
  reportMarker,
  hasReportMarker,
  buildReportIssue,
  staticReply,
  canCreateReport,
};
