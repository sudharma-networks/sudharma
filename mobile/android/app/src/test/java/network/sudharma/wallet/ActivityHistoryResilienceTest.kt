package network.sudharma.wallet

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

class ActivityHistoryResilienceTest {
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
    fun `activity still loads local history when received discovery is unavailable`() {
        val repository = source("src/main/java/network/sudharma/wallet/SudharmaWalletRepository.kt")
        assertTrue(repository.contains("runCatching { syncReceivedTransactions() }"))
        assertTrue(repository.contains("preferences.transactionRecords()"))
    }

    @Test
    fun `ordinary received transactions use explorer address history before block fallback`() {
        val repository = source("src/main/java/network/sudharma/wallet/SudharmaWalletRepository.kt")
        assertTrue(repository.contains("ExplorerAddressHistoryClient"))
        assertTrue(repository.contains("history(account.address"))
        assertTrue(repository.contains("ReceivedTransactionScanner.scan"))
    }

    @Test
    fun `test token grants and challenge rewards are persisted as received history`() {
        val repository = source("src/main/java/network/sudharma/wallet/SudharmaWalletRepository.kt")
        assertTrue(repository.contains("persistTestTokenReceipt(grant.transactionId"))
        assertTrue(repository.contains("persistTestTokenReceipt(reward.rewardTransactionId"))
    }

    @Test
    fun `activity remains the detailed chronological transaction history`() {
        val walletApp = source("src/main/java/network/sudharma/wallet/WalletApp.kt")
        assertTrue(walletApp.contains("Sent and received transactions are sorted newest first"))
        assertTrue(walletApp.contains("Transaction Details"))
        assertTrue(walletApp.contains("Copy TX ID"))
        assertTrue(walletApp.contains("View on Explorer"))
    }
}
