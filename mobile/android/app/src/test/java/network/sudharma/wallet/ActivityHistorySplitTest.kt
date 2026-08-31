package network.sudharma.wallet

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ActivityHistorySplitTest {
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
    fun `history is a separate wallet screen and system back returns home`() {
        assertEquals(WalletScreen.HOME, SystemBackNavigation.previous(WalletScreen.HISTORY))
        assertTrue(SystemBackNavigation.intercepts(WalletScreen.HISTORY))
    }

    @Test
    fun `home exposes separate activity and history destinations`() {
        val walletApp = source("src/main/java/network/sudharma/wallet/WalletApp.kt")
        assertTrue(walletApp.contains("onActivity = { screen = WalletScreen.ACTIVITY }"))
        assertTrue(walletApp.contains("onHistory = { screen = WalletScreen.HISTORY }"))
        assertTrue(walletApp.contains("Text(\"Activity\")"))
        assertTrue(walletApp.contains("Text(\"History\")"))
    }

    @Test
    fun `activity is server diagnostics while history owns transaction records`() {
        val walletApp = source("src/main/java/network/sudharma/wallet/WalletApp.kt")
        assertTrue(walletApp.contains("private fun ServerActivityScreen"))
        assertTrue(walletApp.contains("RPC endpoint"))
        assertTrue(walletApp.contains("Chain height"))
        assertTrue(walletApp.contains("Peers"))
        assertTrue(walletApp.contains("Mempool"))
        assertTrue(walletApp.contains("private fun TransactionHistoryScreen"))
        assertTrue(walletApp.contains("Sent and received transactions are sorted newest first"))
        assertTrue(walletApp.contains("Transaction Details"))
        assertTrue(walletApp.contains("View on Explorer"))
    }
}
