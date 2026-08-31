package network.sudharma.wallet.security

import java.security.MessageDigest
import java.security.SecureRandom
import javax.crypto.SecretKeyFactory
import javax.crypto.spec.PBEKeySpec

object PinVerifier {
    private const val ITERATIONS = 120_000
    private const val KEY_BITS = 256
    private val random = SecureRandom()

    data class PinRecord(
        val version: Int = 1,
        val iterations: Int,
        val saltHex: String,
        val hashHex: String,
    )

    fun create(pin: String): PinRecord {
        validatePin(pin)
        val salt = ByteArray(16).also(random::nextBytes)
        val hash = derive(pin, salt, ITERATIONS)
        return PinRecord(
            iterations = ITERATIONS,
            saltHex = salt.toHex(),
            hashHex = hash.toHex(),
        )
    }

    fun verify(pin: String, record: PinRecord): Boolean {
        if (!pin.matches(Regex("^[0-9]{6}$"))) return false
        if (record.version != 1 || record.iterations < 100_000) return false
        val expected = runCatching { record.hashHex.hexToBytes() }.getOrElse { return false }
        val salt = runCatching { record.saltHex.hexToBytes() }.getOrElse { return false }
        val actual = derive(pin, salt, record.iterations)
        return MessageDigest.isEqual(expected, actual)
    }

    private fun validatePin(pin: String) {
        require(pin.matches(Regex("^[0-9]{6}$"))) { "PIN must contain exactly six digits" }
    }

    private fun derive(pin: String, salt: ByteArray, iterations: Int): ByteArray {
        val spec = PBEKeySpec(pin.toCharArray(), salt, iterations, KEY_BITS)
        return try {
            SecretKeyFactory.getInstance("PBKDF2WithHmacSHA256").generateSecret(spec).encoded
        } finally {
            spec.clearPassword()
        }
    }

    private fun ByteArray.toHex(): String = joinToString("") { "%02x".format(it.toInt() and 0xff) }

    private fun String.hexToBytes(): ByteArray {
        require(length % 2 == 0)
        return chunked(2).map { it.toInt(16).toByte() }.toByteArray()
    }
}
