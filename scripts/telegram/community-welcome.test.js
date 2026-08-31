'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const { staticReply } = require('./community-core.js');

const EXPECTED_WELCOME = [
  '🚀 SUDHARMA NETWORK — PUBLIC TESTNET',
  '',
  '🇮🇳 Built in India. Open to the world. 🌍',
  '',
  'Sudharma Network is an open-source Proof-of-Work blockchain project being built by students from India.',
  '',
  'We are opening our development to people who want to TEST, MINE, BUILD and help us improve.',
  '',
  '🔹 Public Sudharma Testnet',
  '🔹 SUDH test coins',
  '🔹 Android Testnet Wallet',
  '🔹 Khushi GPU Mining',
  '🔹 NVIDIA CUDA testing',
  '🔹 AMD / OpenCL testing',
  '🔹 Open-source development',
  '🔹 Public GitHub development',
  '🔹 Community testing & bug reporting',
  '',
  '👥 WE ARE LOOKING FOR:',
  '',
  '⛏️ GPU miners',
  '📱 Android wallet testers',
  '💻 Developers',
  '🧪 Blockchain/testnet testers',
  '📊 Mining benchmark contributors',
  '🐞 Bug hunters',
  '🌍 Open-source contributors',
  '',
  'This is not an investment promotion.',
  '',
  '⚠️ Sudharma is currently PRE-MAINNET experimental software. Test coins have no promised monetary value. Never give anyone your seed phrase, private key, password or money.',
  '',
  'Our goal is simple:',
  '',
  'USE IT.',
  'MINE IT.',
  'TEST IT.',
  'BREAK IT.',
  'BUILD ON IT.',
  'HELP US MAKE IT BETTER.',
  '',
  '🌐 Website: https://feature-website-foundation.d2mqyt0bt8sl9s.amplifyapp.com/',
  '💬 Community: @sudharma_community',
  '📢 Official updates: @sudharmanetworks',
  '',
  'Sudharma Network',
  'Built in India 🇮🇳',
  'Open to the world 🌍',
].join('\n');

test('welcome reply preserves the approved outreach copy and includes the official website URL', () => {
  assert.equal(staticReply('welcome'), EXPECTED_WELCOME);
});
