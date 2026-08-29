const test = require('node:test');
const assert = require('node:assert/strict');

const {
  parseTelegramMessage,
  isTrustedAssociation,
  publishedMarker,
  hasPublishedMarker,
} = require('./bridge-core.js');

const CONTROL = '<!-- sudharma-telegram-bridge:v1 -->';
const BEGIN = 'TELEGRAM_MESSAGE_BEGIN';
const END = 'TELEGRAM_MESSAGE_END';

function body(message) {
  return `${CONTROL}\n${BEGIN}\n${message}\n${END}`;
}

test('parses one valid block and preserves internal whitespace', () => {
  const message = 'Line one\n\n  indented line  \nLine four';
  assert.equal(parseTelegramMessage(body(message)), message);
});

test('trims only wrapper boundary whitespace around message', () => {
  const input = `${CONTROL}\n${BEGIN}\n\n  hello world  \n\n${END}`;
  assert.equal(parseTelegramMessage(input), 'hello world');
});

test('rejects missing control marker', () => {
  assert.throws(() => parseTelegramMessage(`${BEGIN}\nhello\n${END}`), /control marker/i);
});

test('rejects missing begin marker', () => {
  assert.throws(() => parseTelegramMessage(`${CONTROL}\nhello\n${END}`), /begin marker/i);
});

test('rejects missing end marker', () => {
  assert.throws(() => parseTelegramMessage(`${CONTROL}\n${BEGIN}\nhello`), /end marker/i);
});

test('rejects duplicate control marker', () => {
  assert.throws(() => parseTelegramMessage(`${CONTROL}\n${CONTROL}\n${BEGIN}\nhello\n${END}`), /control marker/i);
});

test('rejects duplicate begin marker', () => {
  assert.throws(() => parseTelegramMessage(`${CONTROL}\n${BEGIN}\n${BEGIN}\nhello\n${END}`), /begin marker/i);
});

test('rejects duplicate end marker', () => {
  assert.throws(() => parseTelegramMessage(`${CONTROL}\n${BEGIN}\nhello\n${END}\n${END}`), /end marker/i);
});

test('rejects markers in the wrong order', () => {
  assert.throws(() => parseTelegramMessage(`${BEGIN}\n${CONTROL}\nhello\n${END}`), /order/i);
});

test('rejects empty message', () => {
  assert.throws(() => parseTelegramMessage(body('   \n\t  ')), /empty/i);
});

test('accepts exactly 4096 Unicode code points', () => {
  const message = '😀'.repeat(4096);
  assert.equal([...parseTelegramMessage(body(message))].length, 4096);
});

test('rejects more than 4096 Unicode code points', () => {
  assert.throws(() => parseTelegramMessage(body('😀'.repeat(4097))), /4096/);
});

test('trusts only GitHub owner member or collaborator associations', () => {
  for (const value of ['OWNER', 'MEMBER', 'COLLABORATOR']) {
    assert.equal(isTrustedAssociation(value), true, value);
  }
  for (const value of ['CONTRIBUTOR', 'FIRST_TIME_CONTRIBUTOR', 'NONE', '', undefined]) {
    assert.equal(isTrustedAssociation(value), false, String(value));
  }
});

test('builds issue-specific published marker', () => {
  assert.equal(publishedMarker(42), '<!-- sudharma-telegram-published:v1 issue=42 -->');
});

test('detects only the exact issue-specific published marker', () => {
  const comments = [
    'normal comment',
    'Published\n<!-- sudharma-telegram-published:v1 issue=41 -->',
    'Success\n<!-- sudharma-telegram-published:v1 issue=42 -->\nmessage_id=123',
  ];
  assert.equal(hasPublishedMarker(comments, 42), true);
  assert.equal(hasPublishedMarker(comments, 4), false);
  assert.equal(hasPublishedMarker(comments, 43), false);
});
