package network.sudharma.wallet

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

// Final 0.1.4 integration verification: this file intentionally exercises user-visible release requirements.
class Wallet014ReleaseRequirementsTest {
    private fun source(path: String): String {
        val candidates = listOf(
            File(path),
            File("app/$path"),
            File("mobile/android/app/$path"),
        )
        return candidates.firstOrNull { it.isFile }?.readText()
            ?: error("Unable to locate source file: $path (user.dir=${System.getProperty("user.dir")})")
    }

    @Test
    fun `wallet 014 keeps the 25 to 50 testnet challenge`() {
        val walletApp = source("src/main/java/network/sudharma/wallet/WalletApp.kt")
        assertTrue(walletApp.contains("Use 25 → 50 Testnet Challenge"))
        assertTrue(walletApp.contains("Check Confirmation & Claim"))
    }

    @Test
    fun `wallet home exposes request test tokens action`() {
        val walletApp = source("src/main/java/network/sudharma/wallet/WalletApp.kt")
        val repository = source("src/main/java/network/sudharma/wallet/SudharmaWalletRepository.kt")
        assertTrue(walletApp.contains("Request Test SUDH"))
        assertTrue(walletApp.contains("requestInitialTestTokens"))
        assertTrue(repository.contains("requestInitialTestTokens"))
    }

    @Test
    fun `transaction ids are selectable copyable and directly linked to explorer`() {
        val walletApp = source("src/main/java/network/sudharma/wallet/WalletApp.kt")
        assertTrue(walletApp.contains("SelectionContainer"))
        assertTrue(walletApp.contains("Copy TX ID"))
        assertTrue(walletApp.contains("ExplorerLinks.transactionUrl(transactionId)"))
    }

    @Test
    fun `wallet version is 014 testnet`() {
        val gradle = source("build.gradle.kts")
        assertTrue(gradle.contains("versionName = \"0.1.4-testnet\""))
    }
}
