'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const {
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
} = require('./community-core.js');

function groupMessage(text = '', extra = {}) {
  return {
    message_id: 44,
    date: 1788048000,
    chat: {
      id: -1001234567890,
      type: 'supergroup',
      username: 'sudharma_community',
    },
    from: {
      id: 987654321,
      first_name: 'Reporter Name',
      username: 'reporter_handle',
    },
    text,
    ...extra,
  };
}

function update(updateId, message) {
  return { update_id: updateId, message };
}

test('exports the pinned official community username', () => {
  assert.equal(OFFICIAL_GROUP_USERNAME, 'sudharma_community');
});

test('accepts only the official public group or supergroup', () => {
  assert.equal(isOfficialChat(groupMessage('/help')), true);
  assert.equal(isOfficialChat({ ...groupMessage('/help'), chat: { id: -1, type: 'group', username: 'sudharma_community' } }), true);
  assert.equal(isOfficialChat({ ...groupMessage('/help'), chat: { id: -1, type: 'supergroup', username: 'other_group' } }), false);
  assert.equal(isOfficialChat({ ...groupMessage('/help'), chat: { id: 1, type: 'private', username: 'sudharma_community' } }), false);
  assert.equal(isOfficialChat(null), false);
});

test('ordinary conversation is ignored', () => {
  assert.deepEqual(
    parseCommunityUpdate(update(1, groupMessage('hello everyone')), 'SudharmaNetworkBot'),
    { kind: 'ignore' },
  );
});

test('private and other-group commands are ignored', () => {
  const privateMessage = { ...groupMessage('/help'), chat: { id: 1, type: 'private', username: 'sudharma_community' } };
  const otherGroupMessage = { ...groupMessage('/help'), chat: { id: -2, type: 'supergroup', username: 'other_group' } };
  assert.equal(parseCommunityUpdate(update(2, privateMessage), 'SudharmaNetworkBot').kind, 'ignore');
  assert.equal(parseCommunityUpdate(update(3, otherGroupMessage), 'SudharmaNetworkBot').kind, 'ignore');
});

test('parses every supported explicit command', () => {
  for (const command of ['help', 'rules', 'testnet', 'miner']) {
    const result = parseCommunityUpdate(update(10, groupMessage(`/${command}`)), 'SudharmaNetworkBot');
    assert.equal(result.kind, 'command');
    assert.equal(result.command, command);
    assert.equal(result.args, '');
    assert.equal(result.message.message_id, 44);
  }

  const report = parseCommunityUpdate(
    update(11, groupMessage('/report miner exits after startup on RX 580 with an OpenCL initialization error')),
    'SudharmaNetworkBot',
  );
  assert.equal(report.kind, 'command');
  assert.equal(report.command, 'report');
  assert.equal(report.args, 'miner exits after startup on RX 580 with an OpenCL initialization error');
});

test('parses commands from captions as well as text', () => {
  const message = groupMessage('', { text: undefined, caption: '/miner' });
  const result = parseCommunityUpdate(update(12, message), 'SudharmaNetworkBot');
  assert.equal(result.kind, 'command');
  assert.equal(result.command, 'miner');
});

test('accepts commands addressed to this bot case-insensitively', () => {
  const result = parseCommunityUpdate(
    update(13, groupMessage('/HELP@sudharmanetworkbot')),
    'SudharmaNetworkBot',
  );
  assert.equal(result.kind, 'command');
  assert.equal(result.command, 'help');
});

test('ignores commands addressed to another bot', () => {
  const result = parseCommunityUpdate(
    update(14, groupMessage('/help@OtherBot')),
    'SudharmaNetworkBot',
  );
  assert.deepEqual(result, { kind: 'ignore' });
});

test('maps an unknown explicit command to help', () => {
  const result = parseCommunityUpdate(update(15, groupMessage('/wat')), 'SudharmaNetworkBot');
  assert.equal(result.kind, 'command');
  assert.equal(result.command, 'help');
});

test('new member service update becomes a generic welcome action without identities', () => {
  const message = groupMessage('', {
    new_chat_members: [{ id: 111222333, first_name: 'Private Name', username: 'private_handle' }],
  });
  const result = parseCommunityUpdate(update(16, message), 'SudharmaNetworkBot');
  assert.equal(result.kind, 'welcome');
  assert.equal(result.message, message);

  const welcome = staticReply('welcome');
  assert.doesNotMatch(welcome, /111222333|Private Name|private_handle/);
  assert.match(welcome, /pre-mainnet|public-testnet/i);
  assert.match(welcome, /https:\/\/feature-website-foundation\.d2mqyt0bt8sl9s\.amplifyapp\.com\//);
});

test('non-message update is ignored', () => {
  assert.deepEqual(parseCommunityUpdate({ update_id: 17, callback_query: {} }, 'SudharmaNetworkBot'), { kind: 'ignore' });
});

test('report length validation uses Unicode code points', () => {
  assert.equal(REPORT_MIN_CODE_POINTS, 20);
  assert.equal(REPORT_MAX_CODE_POINTS, 2000);
  assert.throws(() => validateReportText('😀'.repeat(19)), /at least 20/i);
  assert.equal(validateReportText('😀'.repeat(20)), '😀'.repeat(20));
  assert.equal(validateReportText('x'.repeat(2000)), 'x'.repeat(2000));
  assert.throws(() => validateReportText('x'.repeat(2001)), /no more than 2000/i);
});

test('report validation trims surrounding whitespace and rejects empty input', () => {
  assert.throws(() => validateReportText('   '), /at least 20/i);
  const value = '  miner exits after startup with a reproducible OpenCL error  ';
  assert.equal(validateReportText(value), value.trim());
});

test('neutralizes GitHub mention syntax in untrusted Telegram text', () => {
  assert.equal(
    neutralizeGithubMentions('please ask @alice and @org/team about @everyone'),
    'please ask @\u200balice and @\u200borg/team about @\u200beveryone',
  );
});

test('builds a source link only for an official group message', () => {
  assert.equal(telegramMessageLink(groupMessage('/help')), 'https://t.me/sudharma_community/44');
  const other = { ...groupMessage('/help'), chat: { id: -2, type: 'supergroup', username: 'other_group' } };
  assert.throws(() => telegramMessageLink(other), /official/i);
});

test('builds and finds an exact retry marker', () => {
  assert.equal(reportMarker(123), '<!-- sudharma-telegram-community-report:v1 update_id=123 -->');
  const body = `${reportMarker(123)}\nbody`;
  assert.equal(hasReportMarker(body, 123), true);
  assert.equal(hasReportMarker(body, 124), false);
  assert.equal(hasReportMarker(null, 123), false);
});

test('builds a safe structured report issue without Telegram identity fields', () => {
  const message = groupMessage('/report please ask @alice: miner exits after startup with OpenCL initialization failure');
  const reportText = 'please ask @alice: miner exits after startup with OpenCL initialization failure';
  const built = buildReportIssue({
    updateId: 123,
    reportText,
    message,
    now: new Date('2026-08-30T02:00:00Z'),
  });

  assert.match(built.title, /^Telegram report: /);
  assert.doesNotMatch(built.title, /@alice/);
  assert.match(built.body, /sudharma-telegram-community-report:v1 update_id=123/);
  assert.match(built.body, /https:\/\/t\.me\/sudharma_community\/44/);
  assert.match(built.body, /2026-/);
  assert.match(built.body, /pre-mainnet|public-testnet/i);
  assert.match(built.body, /@\u200balice/);
  assert.doesNotMatch(built.body, /987654321|Reporter Name|reporter_handle/);
});

test('static replies are plain safety-focused text', () => {
  for (const kind of ['help', 'rules', 'testnet', 'miner', 'welcome', 'report-usage', 'throttle']) {
    const reply = staticReply(kind);
    assert.equal(typeof reply, 'string');
    assert.ok(reply.length > 0);
  }
  assert.match(staticReply('help'), /\/report/);
  assert.match(staticReply('report-usage'), /public GitHub/i);
  assert.match(staticReply('report-usage'), /seed phrase|private key|token/i);
  assert.doesNotMatch(staticReply('miner'), /profit|guaranteed return|investment opportunity/i);
  assert.throws(() => staticReply('unknown-kind'), /unknown/i);
});

test('report intake enforces per-run and rolling-hour limits', () => {
  assert.deepEqual(canCreateReport({ createdThisRun: 0, createdLastHour: 0 }), { allowed: true });
  assert.deepEqual(canCreateReport({ createdThisRun: 3, createdLastHour: 0 }), { allowed: false, reason: 'run-limit' });
  assert.deepEqual(canCreateReport({ createdThisRun: 0, createdLastHour: 10 }), { allowed: false, reason: 'hour-limit' });
});
