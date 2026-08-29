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
        val ids = listOf("a".repeat(64), "b".repeat(64))
        val statuses = TransactionActivityLoader.load(ids) { id ->
            TransactionStatus(
                id = id,
                state = if (id.startsWith("a")) TransactionState.PENDING else TransactionState.NOT_FOUND,
            )
        }

        assertEquals(listOf(TransactionState.PENDING, TransactionState.NOT_FOUND), statuses.map { it.state })
    }

    @Test
    fun rpcFailurePropagatesInsteadOfBecomingFailedTransaction() {
        val ids = listOf("a".repeat(64))

        val error = assertThrows(IllegalStateException::class.java) {
            runBlocking {
                TransactionActivityLoader.load(ids) {
                    throw IllegalStateException("network unavailable")
                }
            }
        }

        assertEquals("network unavailable", error.message)
    }
}
