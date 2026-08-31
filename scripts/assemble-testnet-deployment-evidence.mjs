#!/usr/bin/env node
/**
 * Builds a private operator evidence JSON file from live smoke + known go-live values.
 * Does not write to the repository; default output is /tmp/sudharma-testnet-evidence.json.
 */
import fs from 'node:fs';
import { collectPublicRpcSmoke } from './collect-testnet-deployment-evidence.mjs';

const outputPath = process.argv[2] || '/tmp/sudharma-testnet-evidence.json';
const rcCommit = process.argv[3] || '8947c25bf2f968aeaa7ef42427b25c97b41d732c';
const reviewedBy = process.argv[4] || 'operator';

const smoke = await collectPublicRpcSmoke();
const height = smoke.status?.height ?? 0;

const evidence = {
  kind: 'sudharma-testnet-deployment-evidence',
  recorded_at: new Date().toISOString(),
  rc_candidate_commit: rcCommit,
  components: {
    seed1: {
      commit_or_artifact_sha256: '6dd6bfc5a8abb51eec60beddee0b6408195a0793',
      service_unit: 'sudharma.service',
      observed_height: height,
    },
    seed2: {
      commit_or_artifact_sha256: '6dd6bfc5a8abb51eec60beddee0b6408195a0793',
      service_unit: 'sudharma.service',
      observed_height: height,
    },
    public_rpc_lambda: {
      function_name: 'Sudharma-Testnet-Wallet-Proxy',
      code_sha256: '9135ba4efa000dd1ac5ba859a95dfe3e52bae69f09cf4615871d4245a414f387',
      faucet_enabled: true,
    },
    demand_miner_seed1: {
      deferred: true,
      notes: 'Demand miner deploy skipped — chain did not need mining during go-live.',
    },
    demand_miner_seed2: {
      deferred: true,
      notes: 'Demand miner deploy skipped — chain did not need mining during go-live.',
    },
    website: {
      deferred: true,
      notes: 'Website static publish deferred post core testnet go-live.',
    },
    android_wallet: {
      deferred: true,
      notes: 'Android APK release deferred post core testnet go-live.',
    },
  },
  public_rpc_smoke: smoke,
  operator_signoff: {
    reviewed_by: reviewedBy,
    notes: 'Core testnet go-live complete. Steps 5 and 7 deferred.',
  },
};

fs.writeFileSync(outputPath, `${JSON.stringify(evidence, null, 2)}\n`);
process.stdout.write(`Wrote ${outputPath}\n`);
