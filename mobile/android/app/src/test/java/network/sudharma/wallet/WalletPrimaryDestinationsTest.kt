package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

class WalletPrimaryDestinationsTest {
    @Test
    fun `activity and transaction history remain separate home destinations`() {
        assertNotEquals(WalletScreen.ACTIVITY, WalletScreen.HISTORY)
        assertEquals(
            listOf(WalletScreen.ACTIVITY, WalletScreen.HISTORY),
            WalletPrimaryDestinations.activityAndHistory,
        )
    }
}
