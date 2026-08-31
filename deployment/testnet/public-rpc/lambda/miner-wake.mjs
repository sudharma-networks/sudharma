import { proxyWithFailover } from './upstream.mjs';

export async function wakeDemandMiner({ seeds, fetchImpl, timeoutMs }) {
  const request = {
    method: 'POST',
    path: '/v1/miner/wake',
    headers: {},
    body: Buffer.alloc(0),
    queryString: '',
  };
  return proxyWithFailover(request, { seeds, fetchImpl, timeoutMs });
}

export function wakeDemandMinerInBackground(options, logger = console) {
  wakeDemandMiner(options).catch((error) => {
    if (typeof logger?.warn === 'function') {
      logger.warn({
        event: 'demand_miner_wake_failed',
        error: String(error?.message || error),
      });
    }
  });
}
