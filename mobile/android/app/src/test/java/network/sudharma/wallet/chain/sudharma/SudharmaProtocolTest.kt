package network.sudharma.wallet.chain.sudharma

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.math.BigInteger

class SudharmaProtocolTest {
    @Test
    fun privateScalarOneMatchesGoAddressVector() {
        val key = SudharmaCrypto.keyFromPrivateScalar(BigInteger.ONE)
        assertEquals(
            "698bea63dc44a344663ff1429aea10842df27b6b",
            SudharmaCrypto.addressFromPublicKey(key.publicKey),
        )
    }

    @Test
    fun feeAndTransactionIdMatchGoRules() {
        val tx = SudharmaTransaction.create(
            from = "698bea63dc44a344663ff1429aea10842df27b6b",
            to = "00112233445566778899aabbccddeeff00112233",
            amount = 100_000_000L,
            nonce = 7L,
        )
        assertEquals(100_000L, tx.fee)
        assertEquals(
            "b62fd224c08903fdc418aa2ae6040dd3cf5a94b0e9ed0d445952989dc712ddd1",
            tx.id,
        )
    }

    @Test
    fun localSignatureUsesFixedWidthAndVerifies() {
        val key = SudharmaCrypto.keyFromPrivateScalar(BigInteger.ONE)
        val tx = SudharmaTransaction.create(
            from = SudharmaCrypto.addressFromPublicKey(key.publicKey),
            to = "00112233445566778899aabbccddeeff00112233",
            amount = 500_000_000L,
            nonce = 1L,
        )
        val signature = SudharmaCrypto.sign(key.privateScalar, tx.id.toByteArray())
        assertEquals(64, signature.size)
        assertTrue(SudharmaCrypto.verify(key.publicKey, tx.id.toByteArray(), signature))
    }
}
