package network.sudharma.wallet

import network.sudharma.wallet.chain.TransactionState
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Test

class WalletReleaseBaselineTest {
    @Test
    fun `release keeps activity and history as separate navigable destinations`() {
        assertEquals(
            listOf(WalletScreen.ACTIVITY, WalletScreen.HISTORY),
            WalletPrimaryDestinations.activityAndHistory,
        )
        WalletPrimaryDestinations.activityAndHistory.forEach { destination ->
            assertTrue(SystemBackNavigation.intercepts(destination))
            assertEquals(WalletScreen.HOME, SystemBackNavigation.previous(destination))
        }
        assertFalse(SystemBackNavigation.intercepts(WalletScreen.HOME))
    }

    @Test
    fun `release keeps full transaction detail and explorer action`() {
        val transactionId = "b".repeat(64)
        val item = WalletActivityItem(
            record = WalletTransactionRecord(
                id = transactionId,
                direction = TransactionDirection.RECEIVED,
                amountAtomic = 100_000_000L,
                counterparty = "c".repeat(40),
                timestampMs = 1_800_000_000_000L,
            ),
            state = TransactionState.CONFIRMED,
            confirmations = 4,
        )

        val detail = TransactionDetailPresentation.from(item)

        assertEquals(transactionId, detail.transactionId)
        assertEquals("+1.00000000 SUDH", detail.amount)
        assertEquals(4L, detail.confirmations)
        assertEquals(ExplorerLinks.transactionUrl(transactionId), detail.explorerUrl)
    }

    @Test
    fun `release refuses malformed explorer transaction IDs`() {
        listOf("", "ABC", "a".repeat(63), "g".repeat(64), "../wallet").forEach { invalid ->
            assertThrows(IllegalArgumentException::class.java) {
                ExplorerLinks.transactionUrl(invalid)
            }
        }
    }
}
