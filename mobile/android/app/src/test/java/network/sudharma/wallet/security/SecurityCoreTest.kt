package network.sudharma.wallet.security

import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class SecurityCoreTest {
    @Test
    fun encryptedVaultBytesDoNotContainRecoveryPhrase() {
        val key = ByteArray(32) { (it + 1).toByte() }
        val phrase = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
        val encrypted = WalletCipher.encrypt(key, phrase.toByteArray())
        assertFalse(String(encrypted).contains("abandon"))
        assertTrue(phrase.toByteArray().contentEquals(WalletCipher.decrypt(key, encrypted)))
    }

    @Test
    fun pinRecordsUseRandomSaltAndVerify() {
        val first = PinVerifier.create("123456")
        val second = PinVerifier.create("123456")
        assertNotEquals(first.saltHex, second.saltHex)
        assertTrue(PinVerifier.verify("123456", first))
        assertFalse(PinVerifier.verify("654321", first))
    }

    @Test(expected = IllegalArgumentException::class)
    fun pinMustBeSixDigits() {
        PinVerifier.create("12345")
    }
}
