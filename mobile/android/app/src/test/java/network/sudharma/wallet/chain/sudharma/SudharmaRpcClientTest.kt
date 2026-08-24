package network.sudharma.wallet.chain.sudharma

import kotlinx.coroutines.runBlocking
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.math.BigInteger

class SudharmaRpcClientTest {
    private lateinit var server: MockWebServer

    @Before fun setUp() { server = MockWebServer().also { it.start() } }
    @After fun tearDown() { server.shutdown() }

    @Test
    fun statusAndAccountUseExistingV1Endpoints() = runBlocking {
        server.enqueue(MockResponse().setResponseCode(200).setHeader("Content-Type", "application/json")
            .setBody("""{"network":"sudharma","coin":"Sudharma","symbol":"SUDH","node_id":"seed","p2p_address":"x","height":12,"tip_hash":"abc","total_work":"99","peers":2,"mempool":1,"issued_supply":600000000}"""))
        server.enqueue(MockResponse().setResponseCode(200).setHeader("Content-Type", "application/json")
            .setBody("""{"address":"698bea63dc44a344663ff1429aea10842df27b6b","balance":100000000,"confirmed_nonce":3,"next_nonce":4}"""))
        val client = SudharmaRpcClient(server.url("/").toString())
        assertEquals(12L, client.status().height)
        assertEquals(100_000_000L, client.account("698bea63dc44a344663ff1429aea10842df27b6b").balance)
        assertEquals("/v1/status", server.takeRequest().path)
        assertTrue(server.takeRequest().path!!.startsWith("/v1/accounts/"))
    }

    @Test
    fun submitUsesGoTransactionFieldNamesAndBase64Bytes() = runBlocking {
        server.enqueue(MockResponse().setResponseCode(202).setHeader("Content-Type", "application/json")
            .setBody("""{"transaction_id":"abc","relayed_peers":1,"accepted":true}"""))
        val key = SudharmaCrypto.keyFromPrivateScalar(BigInteger.ONE)
        val tx = SudharmaTransaction.create(
            from = SudharmaCrypto.addressFromPublicKey(key.publicKey),
            to = "00112233445566778899aabbccddeeff00112233",
            amount = 100_000_000L,
            nonce = 0L,
        ).signed(BigInteger.ONE)
        val result = SudharmaRpcClient(server.url("/").toString()).submit(tx)
        assertTrue(result.accepted)
        val body = server.takeRequest().body.readUtf8()
        assertTrue(body.contains("\"ID\""))
        assertTrue(body.contains("\"PublicKey\""))
        assertTrue(body.contains("\"Signature\""))
        assertTrue(!body.contains("private"))
    }
}
