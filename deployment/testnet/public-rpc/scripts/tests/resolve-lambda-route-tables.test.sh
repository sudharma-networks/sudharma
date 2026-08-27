#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
subject="${script_dir}/../resolve-lambda-route-tables.sh"
fake_aws="${script_dir}/fixtures/fake-aws"

actual="$(AWS_BIN="$fake_aws" "$subject" vpc-test subnet-explicit subnet-main subnet-explicit)"
expected=$'rtb-explicit\nrtb-main'

if [[ "$actual" != "$expected" ]]; then
  printf 'expected route tables:\n%s\nactual route tables:\n%s\n' "$expected" "$actual" >&2
  exit 1
fi
