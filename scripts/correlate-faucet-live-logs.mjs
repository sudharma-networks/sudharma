#!/usr/bin/env node
import { sanitizeLiveLogText } from './sanitize-faucet-live-logs.mjs';

export function correlateFaucetLiveLogs(text, requestId) {
  const needle = String(requestId || '').trim();
  if (!needle) return [];
  const records = sanitizeLiveLogText(text);
  return records
    .filter((record) => record.request_id === needle)
    .sort((left, right) => String(left.cw_timestamp || '').localeCompare(String(right.cw_timestamp || '')));
}

async function readStdin() {
  const chunks = [];
  for await (const chunk of process.stdin) chunks.push(chunk);
  return Buffer.concat(chunks).toString('utf8');
}

if (process.argv[1]?.endsWith('correlate-faucet-live-logs.mjs')) {
  const requestId = process.argv[2];
  if (!requestId) {
    process.stderr.write('usage: correlate-faucet-live-logs.mjs <request-id> < raw-cloudwatch.txt\n');
    process.exit(2);
  }

  const text = await readStdin();
  for (const record of correlateFaucetLiveLogs(text, requestId)) {
    process.stdout.write(`${JSON.stringify(record)}\n`);
  }
}
