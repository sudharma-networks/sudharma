package network.sudharma.wallet

import network.sudharma.wallet.chain.TransactionStatus

internal object TransactionActivityLoader {
    suspend fun load(
        ids: List<String>,
        fetch: suspend (String) -> TransactionStatus,
    ): List<TransactionStatus> = ids.map { id -> fetch(id) }
}
