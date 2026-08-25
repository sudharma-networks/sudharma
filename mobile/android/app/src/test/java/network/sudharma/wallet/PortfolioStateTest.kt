package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PortfolioStateTest {
    @Test
    fun initialPortfolioIsClearlyLoadingOnSudharmaTestnet() {
        val state = PortfolioState.loading()

        assertTrue(state.loading)
        assertEquals("TESTNET", state.networkBadge)
        assertEquals(listOf("SUDH"), state.assetSymbols)
        assertFalse(state.mainnetEnabled)
        assertFalse(state.swapEnabled)
        assertEquals("Coming later", state.swapLabel)
    }

    @Test
    fun offlineAndRpcErrorsRemainVisibleWithoutInventingABalance() {
        val offline = PortfolioState.offline("Testnet RPC unavailable")

        assertFalse(offline.loading)
        assertEquals(null, offline.balanceAtomic)
        assertEquals("Testnet RPC unavailable", offline.message)
    }

    @Test
    fun loadedPortfolioCarriesAtomicBalanceAndActivity() {
        val loaded = PortfolioState.loaded(
            balanceAtomic = 125_000_000L,
            activity = listOf(ActivitySummary("abc", "PENDING")),
        )

        assertEquals(125_000_000L, loaded.balanceAtomic)
        assertEquals("abc", loaded.activity.single().id)
    }
}
