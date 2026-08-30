#!/usr/bin/env node
import fs from 'node:fs';

const itemPath = process.argv[2];
if (!itemPath) {
  throw new Error('failed-address JSON path is required');
}

const item = JSON.parse(fs.readFileSync(itemPath, 'utf8')) ?? {};
const prepared = Number.parseInt(item.initial_prepared_at || item.initial_reserved_at || '', 10);
const fallback = Number.parseInt(process.env.DIAG_START_MS || '', 10);
const start = Number.isFinite(prepared) ? Math.max(0, prepared - 120_000) : fallback;
if (!Number.isFinite(start)) {
  throw new Error('diagnostic start time is unavailable');
}

if (process.env.GITHUB_ENV) {
  fs.appendFileSync(process.env.GITHUB_ENV, `DIAG_START_MS=${start}\n`);
}

process.stdout.write(`${JSON.stringify({ diag_start_ms: start, used_prepared_at: Number.isFinite(prepared) })}\n`);
