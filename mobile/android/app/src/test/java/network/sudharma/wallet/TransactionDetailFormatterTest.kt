package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class TransactionDetailFormatterTest {
    @Test
    fun labelsReflectDirection() {
        assertEquals("Sent", TransactionDetailFormatter.directionLabel(TransactionDirection.SENT))
        assertEquals("Received", TransactionDetailFormatter.directionLabel(TransactionDirection.RECEIVED))
        assertEquals("Sent to", TransactionDetailFormatter.counterpartyLabel(TransactionDirection.SENT))
        assertEquals("Received from", TransactionDetailFormatter.counterpartyLabel(TransactionDirection.RECEIVED))
    }

    @Test
    fun amountLabelIncludesSignAndUnits() {
        assertEquals("-1.00000000 SUDH", TransactionDetailFormatter.amountLabel(TransactionDirection.SENT, 100_000_000L))
        assertEquals("+0.50000000 SUDH", TransactionDetailFormatter.amountLabel(TransactionDirection.RECEIVED, 50_000_000L))
    }

    @Test
    fun feeLabelOmitsZeroFees() {
        assertNull(TransactionDetailFormatter.feeLabel(0L))
        assertEquals("0.01000000 SUDH", TransactionDetailFormatter.feeLabel(1_000_000L))
    }

    @Test
    fun detectsPlaceholderCounterpartyAndUnknownAmount() {
        val legacy = WalletTransactionRecord(
            id = "a".repeat(64),
            direction = TransactionDirection.SENT,
            amountAtomic = 1L,
            counterparty = PLACEHOLDER_COUNTERPARTY,
            timestampMs = 0L,
        )
        assertFalse(TransactionDetailFormatter.hasKnownCounterparty(legacy.counterparty))
        assertFalse(TransactionDetailFormatter.hasKnownAmount(legacy))
    }

    @Test
    fun detectsKnownCounterpartyAndAmount() {
        val record = WalletTransactionRecord(
            id = "a".repeat(64),
            direction = TransactionDirection.RECEIVED,
            amountAtomic = 100_000_000L,
            counterparty = "b".repeat(40),
            timestampMs = 1_700_000_000_000L,
        )
        assertTrue(TransactionDetailFormatter.hasKnownCounterparty(record.counterparty))
        assertTrue(TransactionDetailFormatter.hasKnownAmount(record))
    }
}
