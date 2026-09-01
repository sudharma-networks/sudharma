package network.sudharma.wallet

import network.sudharma.wallet.chain.sudharma.SudharmaRpcClient
import org.junit.Assert.assertEquals
import org.junit.Test

class TransactionRecordEnricherTest {
    private val wallet = "aa".repeat(20)
    private val sender = "bb".repeat(20)
    private val recipient = "cc".repeat(20)
    private val txId = "dd".repeat(32)

    @Test
    fun enrichesLegacyPlaceholderRecordFromRemote() {
        val legacy = WalletTransactionRecord(
            id = txId,
            direction = TransactionDirection.SENT,
            amountAtomic = 1L,
            counterparty = PLACEHOLDER_COUNTERPARTY,
            timestampMs = 0L,
        )
        val remote = SudharmaRpcClient.RemoteTransaction(
            id = txId,
            from = wallet,
            to = recipient,
            amount = 250_000_000L,
            fee = 10_000_000L,
            nonce = 1L,
        )

        val enriched = TransactionRecordEnricher.enrichOne(legacy, wallet, remote)

        assertEquals(TransactionDirection.SENT, enriched.direction)
        assertEquals(recipient, enriched.counterparty)
        assertEquals(250_000_000L, enriched.amountAtomic)
        assertEquals(10_000_000L, enriched.feeAtomic)
    }

    @Test
    fun enrichesReceivedDirectionFromRemote() {
        val legacy = WalletTransactionRecord(
            id = txId,
            direction = TransactionDirection.SENT,
            amountAtomic = 1L,
            counterparty = PLACEHOLDER_COUNTERPARTY,
            timestampMs = 0L,
        )
        val remote = SudharmaRpcClient.RemoteTransaction(
            id = txId,
            from = sender,
            to = wallet,
            amount = 100_000_000L,
            fee = 5_000_000L,
            nonce = 2L,
        )

        val enriched = TransactionRecordEnricher.enrichOne(legacy, wallet, remote)

        assertEquals(TransactionDirection.RECEIVED, enriched.direction)
        assertEquals(sender, enriched.counterparty)
        assertEquals(100_000_000L, enriched.amountAtomic)
    }

    @Test
    fun leavesCompleteRecordUnchanged() {
        val complete = WalletTransactionRecord(
            id = txId,
            direction = TransactionDirection.SENT,
            amountAtomic = 100_000_000L,
            counterparty = recipient,
            feeAtomic = 5_000_000L,
            timestampMs = 1_700_000_000_000L,
        )
        val remote = SudharmaRpcClient.RemoteTransaction(
            id = txId,
            from = wallet,
            to = recipient,
            amount = 999_000_000L,
            fee = 1_000_000L,
            nonce = 3L,
        )

        assertEquals(complete, TransactionRecordEnricher.enrichOne(complete, wallet, remote))
    }
}
