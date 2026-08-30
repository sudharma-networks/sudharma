package network.sudharma.wallet

import org.junit.Assert.assertTrue
import org.junit.Test

class ExplorerLinksTest {
    @Test
    fun buildsPublicExplorerTransactionUrl() {
        val id = "a".repeat(64)
        assertTrue(ExplorerLinks.transactionUrl(id).contains(id))
        assertTrue(ExplorerLinks.transactionUrl(id).startsWith(ExplorerLinks.PUBLIC_EXPLORER_BASE_URL))
    }
}
