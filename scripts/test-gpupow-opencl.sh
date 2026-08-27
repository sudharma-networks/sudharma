#!/usr/bin/env bash
set -euo pipefail

kernel="compatibility/opencl/gpupow_v1.cl"
harness="compatibility/opencl/verify.c"

for file in "$kernel" "$harness"; do
  if [[ ! -f "$file" ]]; then
    echo "missing OpenCL compatibility file: $file" >&2
    exit 1
  fi
done

cc -std=c11 -Wall -Wextra -Werror \
  $(pkg-config --cflags OpenCL) \
  "$harness" \
  $(pkg-config --libs OpenCL) \
  -o "${RUNNER_TEMP:-/tmp}/sudharma-gpupow-opencl-verify"

"${RUNNER_TEMP:-/tmp}/sudharma-gpupow-opencl-verify" "$kernel"
