package network.sudharma.wallet.backup

import network.sudharma.wallet.security.WalletCipher
import java.security.SecureRandom
import javax.crypto.SecretKeyFactory
import javax.crypto.spec.PBEKeySpec

object EncryptedBackup {
    private const val VERSION: Byte = 1
    private const val SALT_SIZE = 16
    private const val ITERATIONS = 180_000
    private val random = SecureRandom()

    fun encrypt(plaintext: ByteArray, backupPassword: String): ByteArray {
        require(backupPassword.length >= 12) { "backup password must contain at least 12 characters" }
        val salt = ByteArray(SALT_SIZE).also(random::nextBytes)
        val key = derive(backupPassword, salt)
        return try {
            byteArrayOf(VERSION) + salt + WalletCipher.encrypt(key, plaintext)
        } finally {
            key.fill(0)
        }
    }

    fun decrypt(envelope: ByteArray, backupPassword: String): ByteArray {
        require(envelope.size > 1 + SALT_SIZE && envelope[0] == VERSION) { "invalid backup envelope" }
        val salt = envelope.copyOfRange(1, 1 + SALT_SIZE)
        val payload = envelope.copyOfRange(1 + SALT_SIZE, envelope.size)
        val key = derive(backupPassword, salt)
        return try {
            WalletCipher.decrypt(key, payload)
        } finally {
            key.fill(0)
        }
    }

    private fun derive(password: String, salt: ByteArray): ByteArray {
        val spec = PBEKeySpec(password.toCharArray(), salt, ITERATIONS, 256)
        return try {
            SecretKeyFactory.getInstance("PBKDF2WithHmacSHA256").generateSecret(spec).encoded
        } finally {
            spec.clearPassword()
        }
    }
}

interface CloudBackupProvider {
    val configured: Boolean
    suspend fun upload(ciphertext: ByteArray)
    suspend fun download(): ByteArray?
}

object UnconfiguredGoogleBackup : CloudBackupProvider {
    override val configured: Boolean = false
    override suspend fun upload(ciphertext: ByteArray) {
        throw IllegalStateException("Google OAuth/cloud backup is not configured")
    }
    override suspend fun download(): ByteArray? = null
}
