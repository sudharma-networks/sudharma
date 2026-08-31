package network.sudharma.wallet

import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class TestnetFaucetClientTest {
    private lateinit var server: MockWebServer

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun readsDynamicFaucetInfo() = runTest {
        server.enqueue(MockResponse().setResponseCode(200).setBody(
            """{"enabled":true,"challenge_address":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","initial_grant_sudh":100,"challenge_send_sudh":25,"challenge_reward_sudh":50,"max_rounds":5,"cooldown_hours":24,"testnet_only":true}"""
        ))
        val client = TestnetFaucetClient(server.url("/").toString(), OkHttpClient())
        val info = client.info()
        assertTrue(info.enabled)
        assertEquals("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", info.challengeAddress)
        assertEquals(100, info.initialGrantSudh)
        assertEquals(25, info.challengeSendSudh)
        assertEquals(50, info.challengeRewardSudh)
        assertEquals(5, info.maxRounds)
        assertEquals(24, info.cooldownHours)
    }

    @Test
    fun requestsInitialGrantForWalletAddress() = runTest {
        server.enqueue(MockResponse().setResponseCode(202).setBody(
            """{"address":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","amount_sudh":100,"transaction_id":"${"c".repeat(64)}","status":"submitted"}"""
        ))
        val client = TestnetFaucetClient(server.url("/").toString(), OkHttpClient())
        val result = client.requestInitial("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
        assertEquals(100, result.amountSudh)
        assertEquals("c".repeat(64), result.transactionId)
        val request = server.takeRequest()
        assertEquals("/v1/faucet/request", request.path)
        assertTrue(request.body.readUtf8().contains("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"))
    }

    @Test
    fun submitsChallengeClaimWithTransactionId() = runTest {
        server.enqueue(MockResponse().setResponseCode(202).setBody(
            """{"address":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","round":1,"reward_sudh":50,"reward_transaction_id":"${"d".repeat(64)}","next_eligible_at":1700086400000,"status":"submitted"}"""
        ))
        val client = TestnetFaucetClient(server.url("/").toString(), OkHttpClient())
        val result = client.claimChallenge(
            "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
            "c".repeat(64),
        )
        assertEquals(1, result.round)
        assertEquals(50, result.rewardSudh)
        val request = server.takeRequest()
        assertEquals("/v1/faucet/challenge", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"))
        assertTrue(body.contains("c".repeat(64)))
    }
}
