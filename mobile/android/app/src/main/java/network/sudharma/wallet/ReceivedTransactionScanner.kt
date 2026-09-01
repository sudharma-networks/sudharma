package network.sudharma.wallet

import network.sudharma.wallet.chain.sudharma.SudharmaRpcClient

internal object ReceivedTransactionScanner {
    const val DEFAULT_SCAN_DEPTH = 48

    suspend fun scan(
        rpc: SudharmaRpcClient,
        address: String,
        knownIds: Set<String>,
        chainHeight: Long,
        scanDepth: Int = DEFAULT_SCAN_DEPTH,
        nowMs: Long = System.currentTimeMillis(),
    ): List<WalletTransactionRecord> = scanBlocks(
        address = address,
        knownIds = knownIds,
        chainHeight = chainHeight,
        scanDepth = scanDepth,
        nowMs = nowMs,
        fetchBlock = rpc::block,
    )

    internal suspend fun scanBlocks(
        address: String,
        knownIds: Set<String>,
        chainHeight: Long,
        scanDepth: Int = DEFAULT_SCAN_DEPTH,
        nowMs: Long = System.currentTimeMillis(),
        fetchBlock: suspend (Long) -> SudharmaRpcClient.Block,
    ): List<WalletTransactionRecord> {
        require(address.matches(Regex("^[0-9a-f]{40}$"))) { "invalid address" }
        if (chainHeight <= 0L) return emptyList()

        val start = maxOf(1L, chainHeight - scanDepth + 1)
        val discovered = linkedMapOf<String, WalletTransactionRecord>()
        for (height in start..chainHeight) {
            val block = fetchBlock(height)
            for (tx in block.transactions) {
                if (tx.to != address || tx.from == address) continue
                if (knownIds.contains(tx.id) || discovered.containsKey(tx.id)) continue
                discovered[tx.id] = WalletTransactionRecord(
                    id = tx.id,
                    direction = TransactionDirection.RECEIVED,
                    amountAtomic = tx.amount,
                    counterparty = tx.from,
                    feeAtomic = tx.fee,
                    timestampMs = nowMs,
                )
            }
        }
        return discovered.values.toList()
    }
}
