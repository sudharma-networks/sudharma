package network.sudharma.wallet

import network.sudharma.wallet.chain.sudharma.SudharmaPaymentUri
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class WalletBehaviorTest {
    @Test fun parsesEightDecimalSudhAmounts() {
        assertEquals(123_456_789L, SudharmaWalletRepository.parseCoinAmount("1.23456789"))
    }

    @Test(expected = IllegalArgumentException::class)
    fun rejectsMoreThanEightDecimals() {
        SudharmaWalletRepository.parseCoinAmount("1.000000001")
    }

    @Test fun paymentUriRoundTripsTestnetAddress() {
        val address = "00112233445566778899aabbccddeeff00112233"
        assertEquals(address, SudharmaPaymentUri.parse(SudharmaPaymentUri.encode(address)))
        assertEquals(address, SudharmaPaymentUri.parse(address))
        assertNull(SudharmaPaymentUri.parse("ethereum:$address"))
    }
}
