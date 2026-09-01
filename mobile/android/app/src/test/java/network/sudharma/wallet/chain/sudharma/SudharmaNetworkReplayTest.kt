package network.sudharma.wallet.chain.sudharma

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.math.BigInteger

class SudharmaNetworkReplayTest {
    @Test
    fun signaturesAreRejectedAcrossTestnetAndMainnetInBothDirections() {
        val key = SudharmaCrypto.keyFromPrivateScalar(BigInteger.ONE)
        val unsigned = SudharmaTransaction.create(
            from = SudharmaCrypto.addressFromPublicKey(key.publicKey),
            to = "00112233445566778899aabbccddeeff00112233",
            amount = 500_000_000L,
            nonce = 1L,
        )

        val testnetSigned = unsigned.signed(
            key.privateScalar,
            SudharmaSignatureDomain.DEFAULT_NETWORK,
        )
        assertTrue(testnetSigned.verify(SudharmaSignatureDomain.DEFAULT_NETWORK))
        assertFalse(testnetSigned.verify(SudharmaSignatureDomain.MAINNET_NETWORK))

        val mainnetSigned = unsigned.signed(
            key.privateScalar,
            SudharmaSignatureDomain.MAINNET_NETWORK,
        )
        assertTrue(mainnetSigned.verify(SudharmaSignatureDomain.MAINNET_NETWORK))
        assertFalse(mainnetSigned.verify(SudharmaSignatureDomain.DEFAULT_NETWORK))
    }
}
