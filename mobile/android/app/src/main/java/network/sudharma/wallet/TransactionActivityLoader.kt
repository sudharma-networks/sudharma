package network.sudharma.wallet

import network.sudharma.wallet.chain.TransactionStatus

internal object TransactionActivityLoader {
    suspend fun load(
        records: List<WalletTransactionRecord>,
        fetch: suspend (String) -> TransactionStatus,
    ): List<WalletActivityItem> = records
        .sortedByDescending { it.timestampMs }
        .map { record ->
            val status = fetch(record.id)
            WalletActivityItem(record = record, state = status.state, confirmations = status.confirmations)
        }
}
