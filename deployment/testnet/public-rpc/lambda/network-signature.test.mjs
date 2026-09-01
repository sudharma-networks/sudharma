import test from 'node:test';
import assert from 'node:assert/strict';
import { createPublicKey, verify as cryptoVerify } from 'node:crypto';

import { createSigner, COIN } from './faucet.mjs';

const TESTNET_NETWORK_ID = 'sudharma-testnet-1';
const ADDRESS = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';

function publicKeyFromUncompressed(base64PublicKey) {
  const publicKey = Buffer.from(base64PublicKey, 'base64');
  assert.equal(publicKey.length, 65);
  assert.equal(publicKey[0], 0x04);
  return createPublicKey({
    key: {
      kty: 'EC',
      crv: 'P-256',
      x: publicKey.subarray(1, 33).toString('base64url'),
      y: publicKey.subarray(33, 65).toString('base64url'),
    },
    format: 'jwk',
  });
}

function verifies(message, tx) {
  return cryptoVerify(
    'sha256',
    Buffer.from(message, 'utf8'),
    { key: publicKeyFromUncompressed(tx.PublicKey), dsaEncoding: 'ieee-p1363' },
    Buffer.from(tx.Signature, 'base64'),
  );
}

test('faucet signer uses the public-testnet v2 transaction signature domain', () => {
  const signer = createSigner('0'.repeat(63) + '1');
  const tx = signer.signTransaction(ADDRESS, 100 * COIN, 7);

  const boundMessage = `sudharma-tx-v2|${TESTNET_NETWORK_ID}|${tx.ID}`;
  assert.equal(verifies(boundMessage, tx), true, 'network-bound v2 signature did not verify');
  assert.equal(verifies(tx.ID, tx), false, 'legacy tx-id-only signature unexpectedly verified');
});
