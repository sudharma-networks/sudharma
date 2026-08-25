package network.sudharma.wallet.chain.sudharma

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class SudharmaPaymentUriTest {
    private val address = "00112233445566778899aabbccddeeff00112233"

    @Test
    fun versionedTestnetPaymentWithAmountRoundTrips() {
        val encoded = SudharmaPaymentUri.encode(address, amountAtomic = 125_000_000L)
        val payment = SudharmaPaymentUri.decode(encoded)

        assertEquals(address, payment?.address)
        assertEquals(125_000_000L, payment?.amountAtomic)
        assertEquals("testnet", payment?.network)
        assertEquals(1, payment?.version)
    }

    @Test
    fun rawAddressIsAcceptedWithoutInventingAnAmount() {
        val payment = SudharmaPaymentUri.decode(address)

        assertEquals(address, payment?.address)
        assertEquals(null, payment?.amountAtomic)
    }

    @Test
    fun wrongNetworkMalformedAmountAndAddressAreRejected() {
        assertNull(SudharmaPaymentUri.decode("sudharma:$address?network=mainnet&v=1"))
        assertNull(SudharmaPaymentUri.decode("sudharma:$address?network=testnet&v=1&amount=-1"))
        assertNull(SudharmaPaymentUri.decode("sudharma:bad?network=testnet&v=1"))
    }
}
