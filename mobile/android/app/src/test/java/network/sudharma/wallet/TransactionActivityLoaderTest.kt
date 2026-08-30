package network.sudharma.wallet

import kotlinx.coroutines.runBlocking
import network.sudharma.wallet.chain.TransactionState
import network.sudharma.wallet.chain.TransactionStatus
import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

class TransactionActivityLoaderTest {
    @Test
    fun keepsAuthoritativeStatusesUnchanged() = runBlocking {
        val record = WalletTransactionRecord(
            id = "a".repeat(64),
            direction = TransactionDirection.SENT,
            amountAtomic = 100_000_000L,
            counterparty = "b".repeat(40),
        )
        val items = TransactionActivityLoader.load(listOf(record)) { id ->
            TransactionStatus(
                id = id,
                state = if (id.startsWith("a")) TransactionState.PENDING else TransactionState.NOT_FOUND,
            )
        }

        assertEquals(listOf(TransactionState.PENDING), items.map { it.state })
    }

    @Test
    fun rpcFailurePropagatesInsteadOfBecomingFailedTransaction() {
        val record = WalletTransactionRecord(
            id = "a".repeat(64),
            direction = TransactionDirection.SENT,
            amountAtomic = 100_000_000L,
            counterparty = "b".repeat(40),
        )

        val error = assertThrows(IllegalStateException::class.java) {
            runBlocking {
                TransactionActivityLoader.load(listOf(record)) {
                    throw IllegalStateException("network unavailable")
                }
            }
        }

        assertEquals("network unavailable", error.message)
    }
}
