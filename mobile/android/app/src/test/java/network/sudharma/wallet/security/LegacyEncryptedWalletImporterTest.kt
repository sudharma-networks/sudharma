package network.sudharma.wallet.security

import network.sudharma.wallet.chain.sudharma.SudharmaCrypto
import org.bouncycastle.crypto.generators.SCrypt
import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test
import java.math.BigInteger
import java.security.SecureRandom
import java.util.HexFormat
import javax.crypto.Cipher
import javax.crypto.spec.GCMParameterSpec
import javax.crypto.spec.SecretKeySpec

class LegacyEncryptedWalletImporterTest {
    @Test
    fun decryptsGoV1WalletAndPinsTreasuryAddress() {
        val privateScalar = BigInteger("123456789abcdef", 16)
        val json = encryptedFixture(privateScalar, "correct-horse-battery-staple")
        val imported = LegacyEncryptedWalletImporter.importTreasury(
            json.toByteArray(),
            "correct-horse-battery-staple".toCharArray(),
            expectedAddress(privateScalar),
        )
        assertEquals(privateScalar, imported.privateScalar)
        assertEquals(expectedAddress(privateScalar), imported.address)
    }

    @Test
    fun rejectsWrongPasswordWithoutReturningKeyMaterial() {
        val privateScalar = BigInteger("123456789abcdef", 16)
        val json = encryptedFixture(privateScalar, "correct-horse-battery-staple")
        assertThrows(LegacyWalletImportException::class.java) {
            LegacyEncryptedWalletImporter.importTreasury(
                json.toByteArray(),
                "wrong-password-value".toCharArray(),
                expectedAddress(privateScalar),
            )
        }
    }

    @Test
    fun rejectsWalletWhoseDerivedAddressIsNotTreasury() {
        val privateScalar = BigInteger("123456789abcdef", 16)
        val json = encryptedFixture(privateScalar, "correct-horse-battery-staple")
        assertThrows(LegacyWalletAddressMismatchException::class.java) {
            LegacyEncryptedWalletImporter.importTreasury(
                json.toByteArray(),
                "correct-horse-battery-staple".toCharArray(),
                "16d7dc9ec0495109007860a584c7cf9055da9abf",
            )
        }
    }

    @Test
    fun rejectsUnsupportedOrMalformedWalletJson() {
        val malformed = """{"version":2,"kdf":"scrypt","cipher":"aes-256-gcm"}"""
        assertThrows(LegacyWalletImportException::class.java) {
            LegacyEncryptedWalletImporter.importTreasury(
                malformed.toByteArray(),
                "correct-horse-battery-staple".toCharArray(),
                "16d7dc9ec0495109007860a584c7cf9055da9abf",
            )
        }
    }

    private fun expectedAddress(privateScalar: BigInteger): String {
        val key = SudharmaCrypto.keyFromPrivateScalar(privateScalar)
        return SudharmaCrypto.addressFromPublicKey(key.publicKey)
    }

    private fun encryptedFixture(privateScalar: BigInteger, password: String): String {
        val salt = ByteArray(16).also(SecureRandom()::nextBytes)
        val nonce = ByteArray(12).also(SecureRandom()::nextBytes)
        val key = SCrypt.generate(password.toByteArray(), salt, 32768, 8, 1, 32)
        val cipher = Cipher.getInstance("AES/GCM/NoPadding")
        cipher.init(Cipher.ENCRYPT_MODE, SecretKeySpec(key, "AES"), GCMParameterSpec(128, nonce))
        val privateBytes = privateScalar.toByteArray().let {
            if (it.size > 1 && it.first() == 0.toByte()) it.copyOfRange(1, it.size) else it
        }
        val ciphertext = cipher.doFinal(privateBytes)
        key.fill(0)
        return """
            {
              "version": 1,
              "kdf": "scrypt",
              "cipher": "aes-256-gcm",
              "salt": "%s",
              "nonce": "%s",
              "ciphertext": "%s"
            }
        """.trimIndent().format(
            HexFormat.of().formatHex(salt),
            HexFormat.of().formatHex(nonce),
            HexFormat.of().formatHex(ciphertext),
        )
    }
}
