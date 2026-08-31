package network.sudharma.wallet

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

// Regression coverage for Android hardware/gesture back navigation.
class SystemBackNavigationTest {
    private fun source(path: String): String {
        val candidates = listOf(
            File(path),
            File("app/$path"),
            File("mobile/android/app/$path"),
        )
        return candidates.firstOrNull { it.isFile }?.readText()
            ?: error("Unable to locate source file: $path")
    }

    @Test
    fun `system back follows the same previous screen as visible back controls`() {
        val expected = mapOf(
            WalletScreen.RECOVERY to WalletScreen.WELCOME,
            WalletScreen.CONFIRM_RECOVERY to WalletScreen.RECOVERY,
            WalletScreen.IMPORT to WalletScreen.WELCOME,
            WalletScreen.RECEIVE to WalletScreen.HOME,
            WalletScreen.SEND to WalletScreen.HOME,
            WalletScreen.ACTIVITY to WalletScreen.HOME,
            WalletScreen.SETTINGS to WalletScreen.HOME,
            WalletScreen.BACKUP to WalletScreen.SETTINGS,
        )

        expected.forEach { (from, to) ->
            assertTrue("$from should intercept system back", SystemBackNavigation.intercepts(from))
            assertEquals(to, SystemBackNavigation.previous(from))
        }
    }

    @Test
    fun `root screens leave system back to Android`() {
        listOf(
            WalletScreen.SPLASH,
            WalletScreen.WELCOME,
            WalletScreen.SET_PIN,
            WalletScreen.BIOMETRIC_SETUP,
            WalletScreen.UNLOCK,
            WalletScreen.HOME,
        ).forEach { screen ->
            assertFalse("$screen should not trap Android system back", SystemBackNavigation.intercepts(screen))
        }
    }

    @Test
    fun `compose wires system back at wallet and transaction detail levels`() {
        val walletApp = source("src/main/java/network/sudharma/wallet/WalletApp.kt")
        assertTrue(walletApp.contains("BackHandler(enabled = SystemBackNavigation.intercepts(screen))"))
        assertTrue(walletApp.contains("BackHandler(enabled = selected != null)"))
        assertTrue(walletApp.contains("selected = null"))
    }
}
