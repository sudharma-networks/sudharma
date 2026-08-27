# Demand-Based Public Testnet Miner Design

## Status

Approved design for the Sudharma Public Testnet. Mainnet remains disabled.

## Purpose

Automatically confirm valid pending public-testnet transactions through the existing Sudharma Proof-of-Work consensus path. The service mines only when the mempool contains work, so faucet grants and wallet transfers no longer require an operator to run a one-shot workflow.

## Non-Goals

- No consensus, genesis, block reward, difficulty, transaction, or supply-rule changes.
- No premine, arbitrary minting, balance editing, or privileged transaction confirmation.
- No wallet or faucet private keys in the mining service.
- No mining capability inside the faucet or public RPC proxy.
- No Mainnet deployment or activation.
- No continuous empty-block mining.

## Architecture

Run one dedicated systemd service on a single public-testnet operator host. The service uses a small supervisor command in the Sudharma repository and the existing native `sudharmad -mineblocks 1` implementation.

The supervisor polls the host's private loopback RPC status endpoint at a bounded interval. When the reported mempool count is greater than zero, it launches one ephemeral native miner connected to Seed-1 P2P, using an operator-owned public reward address. The miner synchronizes the current chain and mempool, mines exactly one block, broadcasts it, persists its ephemeral working state, and exits.

The supervisor then checks the private RPC until the network observes the new height and a reduced or empty mempool. If valid transactions remain, it may repeat after the configured cooldown. Only one mining child can exist at a time.

The public faucet continues to sign and submit grants only. The Android wallet continues to sign user transfers locally. Neither component can start mining.

## Components

### Demand-Miner Supervisor

A focused command or script with these responsibilities:

- Poll a configured private status URL.
- Validate the status response and mempool count.
- Remain idle when the mempool is empty.
- Start the bounded native miner when work exists.
- Enforce a single child process and a mining cooldown.
- Apply bounded retry/backoff after network or mining failures.
- Emit structured, operator-visible logs without secrets.
- Shut down cleanly on SIGTERM or SIGINT.

It does not inspect, prioritize, create, sign, or modify transactions.

### Native Miner

Reuse the existing `sudharmad` bounded mining mode:

```text
-mineblocks 1
-testmineraddress <operator reward address>
```

The existing mempool validation, block construction, PoW, consensus validation, and P2P broadcast paths remain authoritative.

### systemd Unit

The service runs as a dedicated unprivileged account with restart-on-failure behavior, a bounded restart delay, a private working directory, and hardened systemd settings. It starts only on the public testnet host selected for mining.

Configuration supplies only non-secret operational values:

- Private loopback RPC status URL.
- Seed P2P address.
- Public miner reward address.
- Poll interval, cooldown, and failure backoff.
- Miner binary and ephemeral data paths.

## State and Concurrency

The supervisor keeps only transient operational state. A process lock prevents overlapping miners on the same host. Deployment enables the service on exactly one host initially, preventing Seed-1 and Seed-2 from independently responding to the same pending transaction.

The miner reads canonical chain and mempool state through normal P2P synchronization. It never edits seed-node chain, state, or mempool files directly.

## Transaction Flow

1. A wallet submits a locally signed transaction or the faucet submits a signed test grant.
2. The public RPC validates and relays it to the seed-node mempool.
3. The supervisor observes `mempool > 0` through private RPC.
4. The supervisor launches one bounded miner.
5. The miner synchronizes, constructs a block from valid pending transactions, performs PoW, and broadcasts the block.
6. Seed nodes validate and adopt the block through normal consensus.
7. The supervisor verifies the height transition and mempool reduction, then returns to idle.
8. Wallet balance and activity calls observe the confirmed transaction.

## Safety Controls

- Testnet-only executable/configuration guard; startup fails for any other network identity.
- Private RPC monitoring; no public mining-control endpoint.
- Exactly one block per miner invocation.
- No mining when the mempool is empty.
- Single-process lock and single-host deployment.
- Configured cooldown prevents rapid restart loops.
- Maximum child runtime terminates a stuck miner.
- Bounded HTTP timeouts, response sizes, retries, and log output.
- Reward address must be a valid public Sudharma address; no private key is required.
- Deployment and rollback never modify blockchain data files.

The normal block subsidy increases issued supply only when genuine PoW succeeds. This is expected consensus behavior, not premine or arbitrary minting.

## Failure Handling

- Private RPC unavailable: log the failure, back off, and do not mine.
- Malformed or wrong-network status: fail closed and alert through service logs.
- P2P synchronization failure: child exits; supervisor backs off before retrying.
- Mining timeout: terminate the child, log the result, and retry only after backoff.
- Competing miner confirms first: synchronized mempool becomes empty; bounded miner exits or any valid competing block is handled through normal fork choice.
- Service crash: systemd restarts it after a delay; the process lock prevents overlap.
- Repeated failures: rate-limited logs remain visible to operators while the service avoids a tight loop.

## Deployment and Rollback

Deployment is staged:

1. Add unit tests for polling decisions, locking, backoff, and command construction.
2. Run repository tests, static checks, and secret scans.
3. Install the supervisor and unit on one testnet host without enabling it.
4. Validate configuration and dry-run status polling.
5. Enable the service.
6. Submit one controlled wallet/faucet transaction and verify automatic confirmation.
7. Observe an idle period and prove no empty block is mined.

Rollback stops and disables the systemd service and removes only its unit/configuration and ephemeral working directory. Seed services and blockchain data remain untouched. Any already-mined valid block remains part of normal chain history.

## Verification and Acceptance Criteria

Automated tests must prove:

- Empty mempool produces no miner command.
- Positive mempool produces exactly one bounded miner command.
- Invalid or wrong-network status fails closed.
- Concurrent triggers cannot start a second miner.
- Failures apply bounded backoff.
- Shutdown terminates or waits for the child cleanly.
- Configuration rejects missing/invalid reward addresses and unsafe public control URLs.

Live public-testnet acceptance requires:

- Both seeds begin at the same height and tip.
- A new faucet request enters the mempool.
- The service starts automatically without operator action.
- Exactly one block is mined for that pending batch.
- Both seeds converge on the new height and tip.
- The mempool returns to zero.
- The recipient balance increases by the requested amount.
- Issued supply increases only by the expected 50 SUDH block subsidy.
- With an empty mempool for at least two poll intervals plus one cooldown, height and issued supply remain unchanged.
- Service logs contain no private keys, wallet files, credentials, or signed secret material.

## Operational Notes

The first deployment uses a single miner supervisor for simplicity and deterministic operation. High availability, mining pools, fixed block cadence, fee markets, and Mainnet mining are separate future designs and are outside this implementation.
