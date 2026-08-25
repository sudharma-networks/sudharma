package network.sudharma.wallet.chain.sudharma

import kotlinx.coroutines.runBlocking
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Before
import org.junit.Test
import java.math.BigInteger

class SudharmaTransferFlowTest {
    private lateinit var server: MockWebServer
    private lateinit var adapter: SudharmaChainAdapter

    @Before
    fun setUp() {
        server = MockWebServer().also { it.start() }
        adapter = SudharmaChainAdapter(SudharmaRpcClient(server.url("/").toString()))
    }

    @After fun tearDown() = server.shutdown()

    @Test
    fun acceptedLocallySignedTransferStartsPending() = runBlocking {
        val transfer = signedTransfer()
        server.enqueue(json(202, """{"transaction_id":"${transfer.transactionId}","relayed_peers":1,"accepted":true}"""))

        val status = adapter.submit(transfer)

        assertEquals(transfer.transactionId, status.id)
        assertEquals(network.sudharma.wallet.chain.TransactionState.PENDING, status.state)
    }

    @Test
    fun rpcCannotReplaceLocallySignedTransactionId() {
        val transfer = signedTransfer()
        server.enqueue(json(202, """{"transaction_id":"${"f".repeat(64)}","relayed_peers":1,"accepted":true}"""))

        assertThrows(IllegalArgumentException::class.java) {
            runBlocking { adapter.submit(transfer) }
        }
    }

    @Test
    fun authoritativeStatusDistinguishesConfirmedAndNotFound() = runBlocking {
        val id = "a".repeat(64)
        server.enqueue(json(200, """{"status":"confirmed","block_height":12,"block_hash":"abc","confirmations":3}"""))
        server.enqueue(json(404, """{"error":"transaction not found"}"""))

        val confirmed = adapter.status(id)
        val missing = adapter.status(id)

        assertEquals(network.sudharma.wallet.chain.TransactionState.CONFIRMED, confirmed.state)
        assertEquals(3L, confirmed.confirmations)
        assertEquals(network.sudharma.wallet.chain.TransactionState.NOT_FOUND, missing.state)
    }

    private fun signedTransfer(): network.sudharma.wallet.chain.SignedTransfer {
        val key = SudharmaCrypto.keyFromPrivateScalar(BigInteger.ONE)
        val from = SudharmaCrypto.addressFromPublicKey(key.publicKey)
        val unsigned = adapter.unsigned(
            from = from,
            to = "00112233445566778899aabbccddeeff00112233",
            amountAtomic = 100_000_000L,
            nonce = 7L,
        )
        return adapter.sign(unsigned, key.privateScalar)
    }

    private fun json(code: Int, body: String): MockResponse = MockResponse()
        .setResponseCode(code)
        .setHeader("Content-Type", "application/json")
        .setBody(body)
}
