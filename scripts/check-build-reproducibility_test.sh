#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

require_file() {
  [ -f "$1" ] || fail "$1 is missing"
}

require_file deployment/testnet/public-rpc/lambda/package-lock.json
grep -Fq 'npm ci --ignore-scripts --no-audit --no-fund' .github/workflows/ci.yml \
  || fail 'CI must install Lambda dependencies with npm ci'

require_file mobile/android/gradlew
require_file mobile/android/gradle/wrapper/gradle-wrapper.jar
require_file mobile/android/gradle/wrapper/gradle-wrapper.properties
[ -x mobile/android/gradlew ] || fail 'mobile/android/gradlew must be executable'
grep -Fq 'distributionSha256Sum=544c35d6bd849ae8a5ed0bcea39ba677dc40f49df7d1835561582da2009b961d' mobile/android/gradle/wrapper/gradle-wrapper.properties \
  || fail 'Gradle 8.7 distribution checksum must be pinned'
grep -Fq 'validateDistributionUrl=true' mobile/android/gradle/wrapper/gradle-wrapper.properties \
  || fail 'Gradle wrapper distribution URL validation must remain enabled'
grep -Fq 'run: ./gradlew --no-daemon :app:testDebugUnitTest' .github/workflows/android-wallet.yml \
  || fail 'Android CI unit tests must use the checked-in Gradle wrapper'
grep -Fq 'run: ./gradlew --no-daemon :app:lintDebug' .github/workflows/android-wallet.yml \
  || fail 'Android CI lint must use the checked-in Gradle wrapper'
grep -Fq 'run: ./gradlew --no-daemon :app:assembleDebug' .github/workflows/android-wallet.yml \
  || fail 'Android CI APK build must use the checked-in Gradle wrapper'

require_file web/eslint.config.mjs
node --input-type=module <<'NODE'
import fs from 'node:fs';

const pkg = JSON.parse(fs.readFileSync('web/package.json', 'utf8'));
if (pkg.scripts?.lint !== 'eslint .') {
  throw new Error('web lint script must run eslint . non-interactively');
}
if (!pkg.devDependencies?.eslint || !pkg.devDependencies?.['eslint-config-next']) {
  throw new Error('web must declare eslint and eslint-config-next devDependencies');
}
NODE
grep -Fq 'cache-dependency-path: web/package-lock.json' .github/workflows/website-ci.yml \
  || fail 'Website CI cache must use package-lock.json'
grep -Fq -- '- run: npm ci' .github/workflows/website-ci.yml \
  || fail 'Website CI must install dependencies with npm ci'
grep -Fq -- '- run: npm run lint' .github/workflows/website-ci.yml \
  || fail 'Website CI must run non-interactive lint'

printf 'PASS: clean-checkout build inputs are pinned and CI uses them\n'
