# Task 1 report: strict testnet-only configuration

## Implementation summary

Added the `demandminer` configuration API with strict JSON decoding, fixed public-testnet identity checks, loopback-only status URL validation, lowercase 40-character reward-address validation, absolute path checks, duration parsing, and duration safety constraints. The configuration contains no secret material.

## Files changed

- `demandminer/config.go`
- `demandminer/config_test.go`

## Test commands and results

- `go test ./demandminer -run 'TestConfig' -count=1` — **BLOCKED**: the environment has no Go executable (`/bin/bash: go: command not found`).
- `go test ./demandminer -count=1` — not run because the same missing executable prevents execution.
- `git diff --check` and `git show --check HEAD` — passed.

## TDD evidence

Tests were written before production code. The required RED command was attempted and could not start because `go` is unavailable; therefore no fabricated RED/GREEN test output is reported. Production implementation was then written against the tests and reviewed statically.

## Self-review

Reviewed the committed diff and checked for whitespace errors. The implementation uses `json.Decoder.DisallowUnknownFields`, `url.Parse`, `net.ParseIP`, `filepath.IsAbs`, `time.ParseDuration`, and an anchored lowercase hexadecimal address pattern as required.

## Concerns

Go tests and formatting could not be run in this environment because Go (including `gofmt`) is not installed. A Go-enabled CI or development environment should run the focused and package tests before integration.
