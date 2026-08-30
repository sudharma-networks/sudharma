package network.sudharma.wallet

import network.sudharma.wallet.chain.sudharma.SudharmaRpcClient

internal object TransactionRecordEnricher {
    suspend fun enrich(
        records: List<WalletTransactionRecord>,
        walletAddress: String,
        fetchStatus: suspend (String) -> SudharmaRpcClient.RemoteTransactionStatus,
    ): List<WalletTransactionRecord> = records.map { record ->
        enrichOne(record, walletAddress, fetchStatus)
    }

    internal fun enrichOne(
        record: WalletTransactionRecord,
        walletAddress: String,
        remote: SudharmaRpcClient.RemoteTransaction?,
    ): WalletTransactionRecord {
        if (remote == null) return record
        if (!needsEnrichment(record)) return record

        val direction = when {
            remote.from == walletAddress -> TransactionDirection.SENT
            remote.to == walletAddress -> TransactionDirection.RECEIVED
            else -> record.direction
        }
        val counterparty = when (direction) {
            TransactionDirection.SENT -> remote.to
            TransactionDirection.RECEIVED -> remote.from
        }
        return WalletTransactionRecord(
            id = record.id,
            direction = direction,
            amountAtomic = remote.amount,
            counterparty = counterparty,
            feeAtomic = remote.fee,
            timestampMs = record.timestampMs,
        )
    }

    private suspend fun enrichOne(
        record: WalletTransactionRecord,
        walletAddress: String,
        fetchStatus: suspend (String) -> SudharmaRpcClient.RemoteTransactionStatus,
    ): WalletTransactionRecord {
        if (!needsEnrichment(record)) return record
        return runCatching { fetchStatus(record.id).transaction }
            .getOrNull()
            ?.let { enrichOne(record, walletAddress, it) }
            ?: record
    }

    private fun needsEnrichment(record: WalletTransactionRecord): Boolean =
        !TransactionDetailFormatter.hasKnownCounterparty(record.counterparty)
            || !TransactionDetailFormatter.hasKnownAmount(record)
            || record.timestampMs <= 0L
}
