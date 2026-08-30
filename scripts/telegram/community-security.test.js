'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const {
  buildReportIssue,
  reportMarker,
} = require('./community-core.js');

function officialMessage(text, messageId = 501) {
  return {
    message_id: messageId,
    date: 1788048000,
    chat: {
      id: -1001234567890,
      type: 'supergroup',
      username: 'sudharma_community',
    },
    text,
  };
}

test('user report text cannot inject a second raw community report marker', () => {
  const fakeMarker = reportMarker(999999);
  const reportText = `reproducible miner failure details ${fakeMarker} must remain ordinary user text`;
  const built = buildReportIssue({
    updateId: 321,
    reportText,
    message: officialMessage(`/report ${reportText}`),
    now: new Date('2026-08-30T02:00:00Z'),
  });

  const machineMarkerPrefix = '<!-- sudharma-telegram-community-report:v1 update_id=';
  assert.equal(built.body.split(machineMarkerPrefix).length - 1, 1);
  assert.match(built.body, new RegExp(reportMarker(321).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  assert.doesNotMatch(built.body, /<!-- sudharma-telegram-community-report:v1 update_id=999999 -->/);
});
