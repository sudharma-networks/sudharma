package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class WalletTransactionRecordTest {
    @Test
    fun encodesAndDecodesRoundTrip() {
        val record = WalletTransactionRecord(
            id = "a".repeat(64),
            direction = TransactionDirection.SENT,
            amountAtomic = 100_000_000L,
            counterparty = "b".repeat(40),
            feeAtomic = 10_000_000L,
            timestampMs = 1_700_000_000_000L,
        )
        assertEquals(record, WalletTransactionRecord.decode(record.encode()))
    }

    @Test
    fun rejectsInvalidPayload() {
        assertNull(WalletTransactionRecord.decode("not-a-record"))
    }
}
