package network.sudharma.wallet.recovery

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class RecoveryTest {
    @Test
    fun standardTwelveWordVectorValidates() {
        val phrase = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
        assertTrue(RecoveryPhrase.validate(phrase))
    }

    @Test
    fun mobileDerivationV1IsDeterministic() {
        val phrase = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
        val seed = RecoveryPhrase.seed(phrase)
        val account = SudharmaMobileDerivationV1.derive(seed, 0)
        assertEquals(
            "cda7b93212d7794e06ef0af3835c329f8dc5ababce9428ec62957df54b010203",
            account.privateScalar.toString(16).padStart(64, '0'),
        )
        assertEquals("690ea636b43434befd0ab33e5b97e40d9444b510", account.address)
        assertEquals("sudharma-mobile-v1", account.profile)
    }
}
