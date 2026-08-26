package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class SplashBrandingTest {
    @Test
    fun animatedPresentationUsesPremiumBrandingTimingAndCopy() {
        val presentation = SplashPresentationPolicy.forAnimatorScale(1f)

        assertTrue(presentation.animate)
        assertEquals(1_850L, presentation.delayMillis)
        assertEquals("SUDHARMA", presentation.title)
        assertEquals("TESTNET WALLET", presentation.subtitle)
        assertEquals(900L, presentation.haloPulseMillis)
    }

    @Test
    fun reducedMotionKeepsBrandingButSkipsAnimation() {
        val presentation = SplashPresentationPolicy.forAnimatorScale(0f)

        assertFalse(presentation.animate)
        assertEquals(250L, presentation.delayMillis)
        assertEquals("SUDHARMA", presentation.title)
        assertEquals("TESTNET WALLET", presentation.subtitle)
    }
}
