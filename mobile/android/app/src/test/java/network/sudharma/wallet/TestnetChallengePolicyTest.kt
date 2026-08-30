package network.sudharma.wallet

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TestnetChallengePolicyTest {
    private val address = "0123456789abcdef0123456789abcdef01234567"
    private val info = TestnetFaucetClient.Info(
        enabled = true,
        challengeAddress = address,
        initialGrantSudh = 100,
        challengeSendSudh = 25,
        challengeRewardSudh = 50,
        maxRounds = 5,
        cooldownHours = 24,
    )

    @Test
    fun exactOfficialChallengeMatches() {
        assertTrue(TestnetChallengePolicy.matchesOfficialChallenge(info, address, 2_500_000_000L))
    }

    @Test
    fun wrongRecipientOrAmountNeverMatches() {
        assertFalse(TestnetChallengePolicy.matchesOfficialChallenge(info, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 2_500_000_000L))
        assertFalse(TestnetChallengePolicy.matchesOfficialChallenge(info, address, 2_400_000_000L))
    }

    @Test
    fun disabledOrMissingFaucetNeverMatches() {
        assertFalse(TestnetChallengePolicy.matchesOfficialChallenge(info.copy(enabled = false), address, 2_500_000_000L))
        assertFalse(TestnetChallengePolicy.matchesOfficialChallenge(null, address, 2_500_000_000L))
    }
}
