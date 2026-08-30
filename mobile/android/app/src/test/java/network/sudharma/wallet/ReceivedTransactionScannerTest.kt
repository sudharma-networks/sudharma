package network.sudharma.wallet

import kotlinx.coroutines.runBlocking
import network.sudharma.wallet.chain.sudharma.SudharmaRpcClient
import org.junit.Assert.assertEquals
import org.junit.Test

class ReceivedTransactionScannerTest {
    @Test
    fun discoversIncomingTransactionsForAddress() = runBlocking {
        val address = "c".repeat(40)
        val sender = "d".repeat(40)
        val tx = SudharmaRpcClient.RemoteTransaction(
            id = "e".repeat(64),
            from = sender,
            to = address,
            amount = 50_000_000L,
            fee = 5_000_000L,
            nonce = 0L,
        )

        val found = ReceivedTransactionScanner.scanBlocks(
            address = address,
            knownIds = emptySet(),
            chainHeight = 10L,
            scanDepth = 4,
            nowMs = 123L,
            fetchBlock = { height ->
                SudharmaRpcClient.Block(
                    height = height,
                    transactions = if (height == 10L) listOf(tx) else emptyList(),
                )
            },
        )

        assertEquals(1, found.size)
        assertEquals(TransactionDirection.RECEIVED, found.first().direction)
        assertEquals(sender, found.first().counterparty)
    }
}
