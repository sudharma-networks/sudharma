package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class TestnetChallengeConfigTest {
    @Test
    fun publicTestnetDefaultsAreStable() {
        assertEquals("https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com", TestnetChallengeConfig.DEFAULT_RPC_URL)
        assertEquals("100", TestnetChallengeConfig.INITIAL_GRANT_SUDH)
        assertEquals("25", TestnetChallengeConfig.CHALLENGE_SEND_SUDH)
        assertEquals("50", TestnetChallengeConfig.CHALLENGE_REWARD_SUDH)
        assertEquals(5, TestnetChallengeConfig.MAX_ROUNDS)
        assertEquals(24, TestnetChallengeConfig.COOLDOWN_HOURS)
    }

    @Test
    fun challengeAddressIsUnavailableUntilDedicatedWalletIsProvisioned() {
        assertNull(TestnetChallengeConfig.challengeDepositAddress)
    }
}
