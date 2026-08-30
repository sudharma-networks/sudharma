package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Test

class ExplorerLinksTest {
    @Test
    fun transactionUrlPointsAtPublicWebsiteExplorer() {
        val txid = "a".repeat(64)
        assertEquals(
            "https://feature-website-foundation.d2mqyt0bt8sl9s.amplifyapp.com/explorer/tx?id=$txid",
            ExplorerLinks.transactionUrl(txid),
        )
    }
}
