'use strict';

const CONTROL = '<!-- sudharma-telegram-bridge:v1 -->';
const BEGIN = 'TELEGRAM_MESSAGE_BEGIN';
const END = 'TELEGRAM_MESSAGE_END';
const TRUSTED = new Set(['OWNER', 'MEMBER', 'COLLABORATOR']);
const MAX_CODE_POINTS = 4096;

function countOccurrences(text, needle) {
  if (!needle) return 0;
  let count = 0;
  let offset = 0;
  while (true) {
    const index = text.indexOf(needle, offset);
    if (index === -1) return count;
    count += 1;
    offset = index + needle.length;
  }
}

function requireExactlyOne(body, marker, label) {
  const count = countOccurrences(body, marker);
  if (count !== 1) {
    throw new Error(`Expected exactly one ${label}; found ${count}`);
  }
}

function parseTelegramMessage(body) {
  if (typeof body !== 'string') {
    throw new Error('Issue body must be a string');
  }

  requireExactlyOne(body, CONTROL, 'control marker');
  requireExactlyOne(body, BEGIN, 'begin marker');
  requireExactlyOne(body, END, 'end marker');

  const controlIndex = body.indexOf(CONTROL);
  const beginIndex = body.indexOf(BEGIN);
  const endIndex = body.indexOf(END);
  if (!(controlIndex < beginIndex && beginIndex < endIndex)) {
    throw new Error('Telegram control markers are in the wrong order');
  }

  const messageStart = beginIndex + BEGIN.length;
  const message = body.slice(messageStart, endIndex).trim();
  if (message.length === 0) {
    throw new Error('Telegram message is empty');
  }

  const codePoints = [...message].length;
  if (codePoints > MAX_CODE_POINTS) {
    throw new Error(`Telegram message exceeds ${MAX_CODE_POINTS} Unicode code points`);
  }

  return message;
}

function isTrustedAssociation(value) {
  return TRUSTED.has(value);
}

function publishedMarker(issueNumber) {
  return `<!-- sudharma-telegram-published:v1 issue=${issueNumber} -->`;
}

function hasPublishedMarker(commentBodies, issueNumber) {
  if (!Array.isArray(commentBodies)) return false;
  const marker = publishedMarker(issueNumber);
  return commentBodies.some((body) => typeof body === 'string' && body.includes(marker));
}

module.exports = {
  CONTROL,
  BEGIN,
  END,
  MAX_CODE_POINTS,
  parseTelegramMessage,
  isTrustedAssociation,
  publishedMarker,
  hasPublishedMarker,
};
