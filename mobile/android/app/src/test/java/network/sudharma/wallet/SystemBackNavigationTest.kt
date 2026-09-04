package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class SystemBackNavigationTest {
    @Test
    fun `system back returns each nested screen to its visible parent`() {
        val expectedParents = mapOf(
            WalletScreen.RECOVERY to WalletScreen.WELCOME,
            WalletScreen.CONFIRM_RECOVERY to WalletScreen.RECOVERY,
            WalletScreen.IMPORT to WalletScreen.WELCOME,
            WalletScreen.RECEIVE to WalletScreen.HOME,
            WalletScreen.SEND to WalletScreen.HOME,
            WalletScreen.ACTIVITY to WalletScreen.HOME,
            WalletScreen.HISTORY to WalletScreen.HOME,
            WalletScreen.SETTINGS to WalletScreen.HOME,
            WalletScreen.BACKUP to WalletScreen.SETTINGS,
        )

        expectedParents.forEach { (screen, parent) ->
            assertTrue("$screen must consume Back", SystemBackNavigation.intercepts(screen))
            assertEquals(parent, SystemBackNavigation.previous(screen))
        }
    }

    @Test
    fun `home leaves system back to Android`() {
        assertFalse(SystemBackNavigation.intercepts(WalletScreen.HOME))
        assertEquals(WalletScreen.HOME, SystemBackNavigation.previous(WalletScreen.HOME))
    }
}
