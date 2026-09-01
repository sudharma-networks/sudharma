# Windows GPU miner republish runbook (operator)

Republish the one-click Windows GPU miner zip so miners receive solo + pool launchers (`Start Mining.bat`, `Start Pool Mining.bat`). This does **not** change consensus or seed configuration.

## When to run

- After PR #79 / mining stack changes merge to `main`
- When pool Stratum client or Windows packaging updates ship
- Before pointing miners at a new public testnet pool URL

## Preconditions

- [ ] Green CI on the commit you are publishing from
- [ ] `bash ./scripts/build-windows-gpu-miner.sh` succeeds locally (optional rehearsal)
- [ ] GitHub Actions `contents: write` permission available to workflow

## Publish steps

1. Open GitHub → Actions → **Windows GPU Miner Publish**
2. Click **Run workflow**
3. Set inputs:
   - `tag`: bump semver, e.g. `windows-gpu-miner-v0.1.1`
   - `confirm`: `PUBLISH` (required gate; workflow input `confirm=PUBLISH`)
4. Wait for workflow completion
5. Verify release assets:
   - `sudharma-gpu-miner-windows.zip`
   - `sudharma-gpu-miner-windows.zip.sha256`

## Post-publish verification

```bash
bash ./scripts/probe-testnet-mining-rpc.sh
```

On a Windows test host:

1. Download zip from GitHub Releases (or synced `/downloads/` path after website sync)
2. Extract and run `Start Mining.bat` (solo) with a testnet wallet address
3. Run `Start Pool Mining.bat` only when a public Stratum pool URL exists

## Website sync (optional)

If the site serves same-site download URLs, run the website release sync workflow or update `web/public/data/github-releases.json` through the normal publish path.

## Rollback

- Mark the GitHub release as deprecated in release notes
- Point website download metadata at the previous tag
- Do not delete historical release artifacts without operator policy approval

## Related

- `.github/workflows/windows-gpu-miner-publish.yml`
- `packaging/windows-gpu-miner/`
- `deployment/testnet/pool-operator-runbook.md`
