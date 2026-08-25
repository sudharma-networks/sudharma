package network.sudharma.wallet.recovery

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class RecoveryTest {
    private val standardPhrase =
        "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

    @Test
    fun standardTwelveWordVectorValidates() {
        assertTrue(RecoveryPhrase.validate(standardPhrase))
    }

    @Test
    fun standardTwelveWordVectorProducesOfficialBip39Seed() {
        assertEquals(
            "5eb00bbddcf069084889a8ab9155568165f5c453ccb85e70811aaed6f6da5fc1" +
                "9a5ac40b389cd370d086206dec8aa6c43daea6690f20ad3d8d48b2d2ce9e38e4",
            RecoveryPhrase.seed(standardPhrase).toHex(),
        )
    }

    @Test
    fun mobileDerivationV1IsDeterministic() {
        val seed = RecoveryPhrase.seed(standardPhrase)
        val account = SudharmaMobileDerivationV1.derive(seed, 0)
        assertEquals(
            "cda7b93212d7794e06ef0af3835c329f8dc5ababce9428ec62957df54b010203",
            account.privateScalar.toString(16).padStart(64, '0'),
        )
        assertEquals("690ea636b43434befd0ab33e5b97e40d9444b510", account.address)
        assertEquals("sudharma-mobile-v1", account.profile)
    }

    @Test
    fun accountIndexProducesIndependentDeterministicAccount() {
        val seed = RecoveryPhrase.seed(standardPhrase)
        val first = SudharmaMobileDerivationV1.derive(seed, 0)
        val second = SudharmaMobileDerivationV1.derive(seed, 1)
        assertEquals(
            "122bf0fa8aedc15e4108efc85d4d3d27040cca746de61cd018a70e7b8d9f7b66",
            second.privateScalar.toString(16).padStart(64, '0'),
        )
        assertEquals("107b13b95fe9ddd20472fff6896bf6bee8d89308", second.address)
        assertNotEquals(first.address, second.address)
    }

    @Test
    fun walletMetadataRecordsVersionedDerivationProfile() {
        val metadata = WalletMetadata.sudharmaMobileV1(accountIndex = 0)
        assertEquals("sudharma-mobile", metadata.derivationProfile)
        assertEquals(1, metadata.derivationVersion)
        assertEquals("sudharma-mobile-v1", metadata.profileId)
        assertEquals(0, metadata.accountIndex)
    }

    @Test(expected = IllegalArgumentException::class)
    fun unsupportedDerivationProfileIsRejected() {
        WalletMetadata.restore(
            derivationProfile = "sudharma-mobile",
            derivationVersion = 2,
            accountIndex = 0,
        )
    }

    private fun ByteArray.toHex(): String = joinToString("") { "%02x".format(it.toInt() and 0xff) }
}
