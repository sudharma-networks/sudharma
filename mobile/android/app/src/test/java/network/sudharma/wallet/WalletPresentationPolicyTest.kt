package network.sudharma.wallet

import java.io.File
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

    @Test
    fun automaticTestnetModeHidesManualFaucetAndClaimControls() {
        val presentation = TestnetAutomationPresentationPolicy.automatic()

        assertFalse(presentation.showManualInitialRequest)
        assertFalse(presentation.showManualChallengeClaim)
        assertTrue(presentation.initialFundingMessage.contains("automatically", ignoreCase = true))
        assertTrue(presentation.challengeRewardMessage.contains("automatically", ignoreCase = true))

        val walletSource = File("src/main/java/network/sudharma/wallet/WalletApp.kt").readText()
        assertFalse(walletSource.contains("repository.requestInitialTestTokens()"))
        assertFalse(walletSource.contains("Check Confirmation & Claim"))
    }
}
