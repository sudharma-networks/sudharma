#!/usr/bin/env node
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const ALLOWED_FIELDS = [
  'event',
  'operation',
  'outcome',
  'error_name',
  'http_status',
  'latency_ms',
  'error_category',
  'route',
  'status_code',
  'request_id',
  'cw_timestamp',
];

const ALLOWED_EVENTS = new Set(['faucet_dependency', 'wallet_faucet_error']);
const INSPECT_STRING_KEYS = new Set([
  'event',
  'operation',
  'outcome',
  'error_name',
  'error_category',
  'route',
  'request_id',
]);
const INSPECT_NUMBER_KEYS = new Set(['http_status', 'latency_ms', 'status_code']);

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

export function parseInspectObject(text) {
  const source = String(text || '');
  const start = source.indexOf('{');
  const end = source.lastIndexOf('}');
  if (start < 0 || end <= start) return null;
  const body = source.slice(start, end + 1);
  const record = {};
  for (const key of INSPECT_STRING_KEYS) {
    const match = body.match(new RegExp(`${key}:\\s*'([^']*)'`));
    if (match) record[key] = match[1];
  }
  for (const key of INSPECT_NUMBER_KEYS) {
    const match = body.match(new RegExp(`${key}:\\s*([0-9]+)`));
    if (match) record[key] = Number(match[1]);
  }
  return Object.keys(record).length > 0 ? record : null;
}

const CLOUDWATCH_PREFIX = /^(20\d{2}-\d{2}-\d{2}T[\d:.]+Z)\t([0-9a-f-]{36})\t/i;

export function extractCloudWatchPrefix(text) {
  const match = String(text || '').match(CLOUDWATCH_PREFIX);
  if (!match) return null;
  return { cw_timestamp: match[1], request_id: match[2] };
}

export function sanitizeLiveLogRecord(record, prefix = null) {
  if (!record || typeof record !== 'object' || Array.isArray(record)) return null;
  if (!ALLOWED_EVENTS.has(record.event)) return null;
  const sanitized = {};
  if (prefix?.request_id && !record.request_id) sanitized.request_id = prefix.request_id;
  if (prefix?.cw_timestamp) sanitized.cw_timestamp = prefix.cw_timestamp;
  for (const key of ALLOWED_FIELDS) {
    if (record[key] === undefined || record[key] === null) continue;
    const value = record[key];
    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
      sanitized[key] = value;
    }
  }
  return Object.keys(sanitized).length > 0 ? sanitized : null;
}

function splitLogChunks(text) {
  const source = String(text || '');
  const parts = source.split(/(?=20\d{2}-\d{2}-\d{2}T[\d:.]+Z\t)/);
  return parts.filter((part) => part.trim());
}

export function sanitizeLiveLogText(text) {
  const records = [];
  const seen = new Set();
  function push(parsed, prefix = null) {
    const sanitized = sanitizeLiveLogRecord(parsed, prefix);
    if (!sanitized) return;
    const key = JSON.stringify(sanitized);
    if (seen.has(key)) return;
    seen.add(key);
    records.push(sanitized);
  }

  for (const line of String(text || '').split(/\r?\n/)) {
    if (!line.trim()) continue;
    push(extractJsonObject(line), extractCloudWatchPrefix(line));
  }
  for (const chunk of splitLogChunks(text)) {
    const prefix = extractCloudWatchPrefix(chunk);
    push(extractJsonObject(chunk) || parseInspectObject(chunk), prefix);
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
