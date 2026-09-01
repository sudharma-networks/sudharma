# Mempool and transaction resource bounds

## Policy decisions

- **Canonical addresses:** `From` and `To` must be exactly 40 lowercase hexadecimal characters.
- **Dust minimum:** transfers below `1000` atomic units are rejected because the configured 0.10% fee floors to zero.
- **Mempool capacity:** at most `4096` transactions and `4 MiB` estimated serialized bytes.
- **Block limits:** at most `1000` transactions and `1 MiB` serialized transaction payload per block.

## Admission path

1. `ValidateResourceBounds` rejects malformed/oversized transactions before replay.
2. Mempool capacity is checked before expensive pending-set replay.
3. Block validation rejects blocks that exceed transaction count/byte limits.

## Remaining work

Incremental mempool indexing to avoid O(n) replay cost per candidate remains a follow-up optimization once baseline bounds are live.
