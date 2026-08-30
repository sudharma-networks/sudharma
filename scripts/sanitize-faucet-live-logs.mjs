#!/usr/bin/env node
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const ALLOWED_FIELDS = [
  'event',
  'operation',
  'outcome',
  'error_name',
  'http_status',
  'error_category',
  'latency_ms',
  'route',
  'status_code',
  'request_id',
];

const ALLOWED_EVENTS = new Set(['faucet_dependency', 'wallet_faucet_error']);

export function extractJsonObject(text) {
  const source = String(text || '');
  const start = source.indexOf('{');
  const end = source.lastIndexOf('}');
  if (start < 0 || end <= start) return null;
  try {
    return JSON.parse(source.slice(start, end + 1));
  } catch {
    return null;
  }
}

export function sanitizeLiveLogRecord(record) {
  if (!record || typeof record !== 'object' || Array.isArray(record)) return null;
  if (!ALLOWED_EVENTS.has(record.event)) return null;
  const sanitized = {};
  for (const key of ALLOWED_FIELDS) {
    if (record[key] === undefined || record[key] === null) continue;
    const value = record[key];
    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
      sanitized[key] = value;
    }
  }
  return Object.keys(sanitized).length > 0 ? sanitized : null;
}

export function sanitizeLiveLogText(text) {
  const lines = String(text || '').split(/\r?\n/);
  const records = [];
  for (const line of lines) {
    if (!line.trim()) continue;
    const parsed = extractJsonObject(line);
    const sanitized = sanitizeLiveLogRecord(parsed);
    if (sanitized) records.push(sanitized);
  }
  return records;
}

async function readStdin() {
  const chunks = [];
  for await (const chunk of process.stdin) chunks.push(chunk);
  return Buffer.concat(chunks).toString('utf8');
}

if (path.resolve(fileURLToPath(import.meta.url)) === path.resolve(process.argv[1] || '')) {
  const text = await readStdin();
  for (const record of sanitizeLiveLogText(text)) {
    process.stdout.write(`${JSON.stringify(record)}\n`);
  }
}
