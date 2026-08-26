package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Test

class WalletPreferencesDefaultRpcTest {
    @Test
    fun `default rpc is public Sudharma Testnet endpoint`() {
        assertEquals(
            "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com",
            WalletPreferences.DEFAULT_RPC_URL,
        )
    }
}
