package network.sudharma.wallet.security

import com.squareup.moshi.JsonClass
import com.squareup.moshi.Moshi
import com.squareup.moshi.kotlin.reflect.KotlinJsonAdapterFactory
import network.sudharma.wallet.chain.sudharma.SudharmaCrypto
import org.bouncycastle.crypto.generators.SCrypt
import java.math.BigInteger
import java.nio.CharBuffer
import java.nio.charset.CodingErrorAction
import java.nio.charset.StandardCharsets
import javax.crypto.AEADBadTagException
import javax.crypto.Cipher
import javax.crypto.spec.GCMParameterSpec
import javax.crypto.spec.SecretKeySpec

open class LegacyWalletImportException(message: String, cause: Throwable? = null) :
    IllegalArgumentException(message, cause)

class LegacyWalletAddressMismatchException :
    LegacyWalletImportException("The selected wallet is not the Sudharma development treasury.")

data class ImportedTreasuryWallet(
    val privateScalar: BigInteger,
    val address: String,
)

@JsonClass(generateAdapter = false)
private data class GoEncryptedWalletV1(
    val version: Int,
    val kdf: String,
    val cipher: String,
    val salt: String,
    val nonce: String,
    val ciphertext: String,
)

object LegacyEncryptedWalletImporter {
    private const val MAX_JSON_BYTES = 64 * 1024
    private const val SCRYPT_N = 32768
    private const val SCRYPT_R = 8
    private const val SCRYPT_P = 1
    private const val KEY_BYTES = 32
    private const val SALT_BYTES = 16
    private const val NONCE_BYTES = 12
    private const val GCM_TAG_BITS = 128
    private val lowerHex = Regex("^[0-9a-f]+$")
    private val adapter = Moshi.Builder()
        .add(KotlinJsonAdapterFactory())
        .build()
        .adapter(GoEncryptedWalletV1::class.java)

    fun importTreasury(
        jsonBytes: ByteArray,
        password: CharArray,
        expectedAddress: String,
    ): ImportedTreasuryWallet {
        requireExpectedAddress(expectedAddress)
        if (jsonBytes.isEmpty() || jsonBytes.size > MAX_JSON_BYTES) {
            throw LegacyWalletImportException("Invalid encrypted wallet file.")
        }
        if (password.isEmpty()) {
            throw LegacyWalletImportException("Wallet password is required.")
        }

        val stored = try {
            adapter.fromJson(jsonBytes.toString(StandardCharsets.UTF_8))
        } catch (error: Exception) {
            throw LegacyWalletImportException("Invalid encrypted wallet file.", error)
        } ?: throw LegacyWalletImportException("Invalid encrypted wallet file.")

        if (stored.version != 1 || stored.kdf != "scrypt" || stored.cipher != "aes-256-gcm") {
            throw LegacyWalletImportException("Unsupported encrypted wallet format.")
        }

        val salt = decodeHex(stored.salt, SALT_BYTES)
        val nonce = decodeHex(stored.nonce, NONCE_BYTES)
        val ciphertext = decodeHex(stored.ciphertext, expectedBytes = null)
        if (ciphertext.size <= GCM_TAG_BITS / 8) {
            throw LegacyWalletImportException("Invalid encrypted wallet file.")
        }

        val passwordBytes = passwordBytes(password)
        val key = try {
            SCrypt.generate(passwordBytes, salt, SCRYPT_N, SCRYPT_R, SCRYPT_P, KEY_BYTES)
        } catch (error: Exception) {
            passwordBytes.fill(0)
            throw LegacyWalletImportException("Unable to unlock encrypted wallet.", error)
        }
        passwordBytes.fill(0)

        val privateBytes = try {
            val cipher = Cipher.getInstance("AES/GCM/NoPadding")
            cipher.init(Cipher.DECRYPT_MODE, SecretKeySpec(key, "AES"), GCMParameterSpec(GCM_TAG_BITS, nonce))
            cipher.doFinal(ciphertext)
        } catch (error: AEADBadTagException) {
            throw LegacyWalletImportException("Incorrect wallet password or damaged wallet file.")
        } catch (error: Exception) {
            throw LegacyWalletImportException("Unable to unlock encrypted wallet.", error)
        } finally {
            key.fill(0)
            salt.fill(0)
            nonce.fill(0)
            ciphertext.fill(0)
        }

        return try {
            val privateScalar = BigInteger(1, privateBytes)
            val keyMaterial = try {
                SudharmaCrypto.keyFromPrivateScalar(privateScalar)
            } catch (error: Exception) {
                throw LegacyWalletImportException("Encrypted wallet contains an invalid private key.", error)
            }
            val address = SudharmaCrypto.addressFromPublicKey(keyMaterial.publicKey)
            if (address != expectedAddress) throw LegacyWalletAddressMismatchException()
            ImportedTreasuryWallet(privateScalar, address)
        } finally {
            privateBytes.fill(0)
        }
    }

    private fun requireExpectedAddress(address: String) {
        if (address.length != 40 || !lowerHex.matches(address)) {
            throw LegacyWalletImportException("Invalid configured treasury address.")
        }
    }

    private fun decodeHex(value: String, expectedBytes: Int?): ByteArray {
        if (value.isEmpty() || value.length % 2 != 0 || !lowerHex.matches(value)) {
            throw LegacyWalletImportException("Invalid encrypted wallet file.")
        }
        val decoded = ByteArray(value.length / 2)
        for (index in decoded.indices) {
            val high = value[index * 2].digitToIntOrNull(16)
            val low = value[index * 2 + 1].digitToIntOrNull(16)
            if (high == null || low == null) {
                decoded.fill(0)
                throw LegacyWalletImportException("Invalid encrypted wallet file.")
            }
            decoded[index] = ((high shl 4) or low).toByte()
        }
        if (expectedBytes != null && decoded.size != expectedBytes) {
            decoded.fill(0)
            throw LegacyWalletImportException("Invalid encrypted wallet file.")
        }
        return decoded
    }

    private fun passwordBytes(password: CharArray): ByteArray {
        val encoder = StandardCharsets.UTF_8.newEncoder()
            .onMalformedInput(CodingErrorAction.REPORT)
            .onUnmappableCharacter(CodingErrorAction.REPORT)
        val encoded = try {
            encoder.encode(CharBuffer.wrap(password))
        } catch (error: Exception) {
            throw LegacyWalletImportException("Wallet password contains invalid text.", error)
        }
        val result = ByteArray(encoded.remaining())
        encoded.get(result)
        if (encoded.hasArray()) encoded.array().fill(0)
        return result
    }
}
