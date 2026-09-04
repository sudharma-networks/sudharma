package network.sudharma.wallet

import kotlinx.coroutines.test.runTest
import network.sudharma.wallet.chain.TransactionState
import network.sudharma.wallet.chain.TransactionStatus
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ActivityDetailNavigationTest {
    private val older = WalletTransactionRecord(
        id = "1".repeat(64),
        direction = TransactionDirection.SENT,
        amountAtomic = 25_00000000L,
        counterparty = "a".repeat(40),
        feeAtomic = 1000L,
        timestampMs = 1_700_000_000_000L,
    )
    private val newer = WalletTransactionRecord(
        id = "2".repeat(64),
        direction = TransactionDirection.RECEIVED,
        amountAtomic = 50_00000000L,
        counterparty = "b".repeat(40),
        timestampMs = 1_800_000_000_000L,
    )

    @Test
    fun `activity history is sorted newest transaction first`() = runTest {
        val items = TransactionActivityLoader.load(listOf(older, newer)) { id ->
            TransactionStatus(id, TransactionState.CONFIRMED, confirmations = 3)
        }

        assertEquals(listOf(newer.id, older.id), items.map { it.record.id })
    }

    @Test
    fun `transaction detail presentation keeps full verification data`() {
        val item = WalletActivityItem(newer, TransactionState.CONFIRMED, confirmations = 7)
        val detail = TransactionDetailPresentation.from(item)

        assertEquals("Received", detail.direction)
        assertEquals(newer.id, detail.transactionId)
        assertEquals(newer.counterparty, detail.counterparty)
        assertEquals(7L, detail.confirmations)
        assertEquals(ExplorerLinks.transactionUrl(newer.id), detail.explorerUrl)
        assertTrue(detail.dateTime.isNotBlank())
    }
}
