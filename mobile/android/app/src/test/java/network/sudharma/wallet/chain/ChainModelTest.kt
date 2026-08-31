package network.sudharma.wallet.chain

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ChainModelTest {
    @Test
    fun networkIdentityDistinguishesTestnetAndMainnet() {
        val testnet = NetworkId("sudharma", "testnet", isTestnet = true)
        val mainnet = NetworkId("sudharma", "mainnet", isTestnet = false)
        assertTrue(testnet.isTestnet)
        assertFalse(mainnet.isTestnet)
        assertEquals("sudharma:testnet", testnet.key)
    }

    @Test
    fun atomicAmountFormatsEightDecimals() {
        val amount = AssetAmount(symbol = "SUDH", atomic = 123_456_789L, decimals = 8)
        assertEquals("1.23456789", amount.formatted())
    }

    @Test(expected = IllegalArgumentException::class)
    fun amountRejectsNegativeAtomicValue() {
        AssetAmount(symbol = "SUDH", atomic = -1L, decimals = 8)
    }
}
