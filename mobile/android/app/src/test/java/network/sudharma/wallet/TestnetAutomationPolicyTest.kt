package network.sudharma.wallet

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TestnetAutomationPolicyTest {
    @Test
    fun `fresh eligible wallet automatically requests initial test grant`() {
        assertTrue(
            TestnetAutomationPolicy.shouldRequestInitialGrant(
                faucetEnabled = true,
                balanceAtomic = 0L,
                requestAlreadyAttempted = false,
            ),
        )
    }

    @Test
    fun `initial grant is not repeatedly requested in same home session`() {
        assertFalse(
            TestnetAutomationPolicy.shouldRequestInitialGrant(
                faucetEnabled = true,
                balanceAtomic = 0L,
                requestAlreadyAttempted = true,
            ),
        )
    }

    @Test
    fun `funded wallet does not automatically request initial grant`() {
        assertFalse(
            TestnetAutomationPolicy.shouldRequestInitialGrant(
                faucetEnabled = true,
                balanceAtomic = 1L,
                requestAlreadyAttempted = false,
            ),
        )
    }

    @Test
    fun `confirmed challenge automatically starts reward claim`() {
        assertTrue(
            TestnetAutomationPolicy.shouldClaimChallengeReward(
                challengeMode = true,
                transactionConfirmed = true,
                claimAlreadyAttempted = false,
            ),
        )
    }

    @Test
    fun `unconfirmed challenge waits without claiming`() {
        assertFalse(
            TestnetAutomationPolicy.shouldClaimChallengeReward(
                challengeMode = true,
                transactionConfirmed = false,
                claimAlreadyAttempted = false,
            ),
        )
    }

    @Test
    fun `normal transfer never triggers challenge reward claim`() {
        assertFalse(
            TestnetAutomationPolicy.shouldClaimChallengeReward(
                challengeMode = false,
                transactionConfirmed = true,
                claimAlreadyAttempted = false,
            ),
        )
    }
}
