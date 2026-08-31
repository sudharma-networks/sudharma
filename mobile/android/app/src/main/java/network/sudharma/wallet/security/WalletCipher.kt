package network.sudharma.wallet.security

import java.security.SecureRandom
import javax.crypto.Cipher
import javax.crypto.spec.GCMParameterSpec
import javax.crypto.spec.SecretKeySpec

object WalletCipher {
    private const val VERSION: Byte = 1
    private const val NONCE_SIZE = 12
    private const val TAG_BITS = 128
    private val random = SecureRandom()

    fun encrypt(key: ByteArray, plaintext: ByteArray): ByteArray {
        require(key.size == 32) { "wallet key must be 32 bytes" }
        val nonce = ByteArray(NONCE_SIZE).also(random::nextBytes)
        val cipher = Cipher.getInstance("AES/GCM/NoPadding")
        cipher.init(
            Cipher.ENCRYPT_MODE,
            SecretKeySpec(key, "AES"),
            GCMParameterSpec(TAG_BITS, nonce),
        )
        val ciphertext = cipher.doFinal(plaintext)
        return byteArrayOf(VERSION) + nonce + ciphertext
    }

    fun decrypt(key: ByteArray, envelope: ByteArray): ByteArray {
        require(key.size == 32) { "wallet key must be 32 bytes" }
        require(envelope.size > 1 + NONCE_SIZE) { "invalid wallet envelope" }
        require(envelope[0] == VERSION) { "unsupported wallet envelope version" }
        val nonce = envelope.copyOfRange(1, 1 + NONCE_SIZE)
        val ciphertext = envelope.copyOfRange(1 + NONCE_SIZE, envelope.size)
        val cipher = Cipher.getInstance("AES/GCM/NoPadding")
        cipher.init(
            Cipher.DECRYPT_MODE,
            SecretKeySpec(key, "AES"),
            GCMParameterSpec(TAG_BITS, nonce),
        )
        return cipher.doFinal(ciphertext)
    }
}
