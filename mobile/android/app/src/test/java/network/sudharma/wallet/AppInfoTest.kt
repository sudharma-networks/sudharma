package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Test

class AppInfoTest {
    @Test
    fun walletIdentityIsSudharmaTestnet() {
        assertEquals("Sudharma Wallet", AppInfo.NAME)
        assertEquals("Sudharma Testnet", AppInfo.NETWORK)
    }
}
