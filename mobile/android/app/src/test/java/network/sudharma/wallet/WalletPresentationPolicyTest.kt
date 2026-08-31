package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class WalletPresentationPolicyTest {
    @Test
    fun reducedMotionDisablesSplashAnimationAndShortensDelay() {
        val policy = SplashPresentationPolicy.forAnimatorScale(0f)

        assertFalse(policy.animate)
        assertEquals(250L, policy.delayMillis)
    }

    @Test
    fun normalSplashRemainsBelowTwoPointFiveSeconds() {
        val policy = SplashPresentationPolicy.forAnimatorScale(1f)

        assertTrue(policy.animate)
        assertTrue(policy.delayMillis <= 2_500L)
    }

    @Test
    fun everyRecoveryPhraseEntryOrDisplayScreenIsSensitive() {
        assertTrue(WalletPresentationPolicy.isSensitive(WalletScreen.RECOVERY))
        assertTrue(WalletPresentationPolicy.isSensitive(WalletScreen.CONFIRM_RECOVERY))
        assertTrue(WalletPresentationPolicy.isSensitive(WalletScreen.IMPORT))
        assertTrue(WalletPresentationPolicy.isSensitive(WalletScreen.BACKUP))
        assertFalse(WalletPresentationPolicy.isSensitive(WalletScreen.HOME))
    }
}
