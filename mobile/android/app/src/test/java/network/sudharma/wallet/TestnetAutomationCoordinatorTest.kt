package network.sudharma.wallet

import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class TestnetAutomationCoordinatorTest {
    @Test
    fun `eligible zero balance wallet requests initial grant automatically`() = runBlocking {
        var requests = 0
        val coordinator = TestnetAutomationCoordinator(
            walletReady = { true },
            faucetEnabled = { true },
            balanceAtomic = { 0L },
            requestInitial = { requests += 1 },
            pendingChallengeId = { null },
            transactionConfirmed = { false },
            claimChallenge = {},
            clearPendingChallenge = {},
        )

        coordinator.tick()

        assertEquals(1, requests)
    }

    @Test
    fun `confirmed pending challenge is claimed and cleared automatically`() = runBlocking {
        var pending: String? = "a".repeat(64)
        var claimed: String? = null
        val coordinator = TestnetAutomationCoordinator(
            walletReady = { true },
            faucetEnabled = { true },
            balanceAtomic = { 100L },
            requestInitial = {},
            pendingChallengeId = { pending },
            transactionConfirmed = { true },
            claimChallenge = { claimed = it },
            clearPendingChallenge = { pending = null },
        )

        coordinator.tick()

        assertEquals("a".repeat(64), claimed)
        assertNull(pending)
    }

    @Test
    fun `unconfirmed pending challenge is left for a later tick`() = runBlocking {
        var pending: String? = "b".repeat(64)
        var claims = 0
        val coordinator = TestnetAutomationCoordinator(
            walletReady = { true },
            faucetEnabled = { true },
            balanceAtomic = { 100L },
            requestInitial = {},
            pendingChallengeId = { pending },
            transactionConfirmed = { false },
            claimChallenge = { claims += 1 },
            clearPendingChallenge = { pending = null },
        )

        coordinator.tick()

        assertEquals(0, claims)
        assertEquals("b".repeat(64), pending)
    }

    @Test
    fun `failed pending challenge is cleared so a clean retry is possible`() = runBlocking {
        var pending: String? = "c".repeat(64)
        var claims = 0
        val coordinator = TestnetAutomationCoordinator(
            walletReady = { true },
            faucetEnabled = { true },
            balanceAtomic = { 100L },
            requestInitial = {},
            pendingChallengeId = { pending },
            transactionConfirmed = { false },
            transactionFailed = { true },
            claimChallenge = { claims += 1 },
            clearPendingChallenge = { pending = null },
        )

        coordinator.tick()

        assertEquals(0, claims)
        assertNull(pending)
    }
}
