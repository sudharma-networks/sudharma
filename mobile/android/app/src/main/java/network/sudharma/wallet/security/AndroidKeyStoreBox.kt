package network.sudharma.wallet.security

import android.os.Build
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.security.keystore.StrongBoxUnavailableException
import androidx.annotation.RequiresApi
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

object AndroidKeyStoreBox {
    private const val KEYSTORE = "AndroidKeyStore"
    private const val ALIAS = "sudharma-wallet-wrap-v1"
    private const val VERSION: Byte = 1
    private const val IV_SIZE = 12

    private fun key(): SecretKey {
        val store = KeyStore.getInstance(KEYSTORE).apply { load(null) }
        (store.getKey(ALIAS, null) as? SecretKey)?.let { return it }

        return if (StrongBoxPolicy.shouldAttemptStrongBox(Build.VERSION.SDK_INT)) {
            generateStrongBoxKeyWithFallback()
        } else {
            generateKey(useStrongBox = false)
        }
    }

    @RequiresApi(28)
    private fun generateStrongBoxKeyWithFallback(): SecretKey =
        try {
            generateKey(useStrongBox = true)
        } catch (_: StrongBoxUnavailableException) {
            generateKey(useStrongBox = false)
        }

    private fun generateKey(useStrongBox: Boolean): SecretKey {
        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, KEYSTORE)
        val builder = KeyGenParameterSpec.Builder(
            ALIAS,
            KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT,
        )
            .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
            .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
            .setKeySize(256)
            .setRandomizedEncryptionRequired(true)

        if (useStrongBox && Build.VERSION.SDK_INT >= 28) {
            builder.setIsStrongBoxBacked(true)
        }

        generator.init(builder.build())
        return generator.generateKey()
    }

    fun wrap(data: ByteArray): ByteArray {
        val cipher = Cipher.getInstance("AES/GCM/NoPadding")
        cipher.init(Cipher.ENCRYPT_MODE, key())
        val iv = cipher.iv
        require(iv.size == IV_SIZE)
        return byteArrayOf(VERSION) + iv + cipher.doFinal(data)
    }

    fun unwrap(envelope: ByteArray): ByteArray {
        require(envelope.size > 1 + IV_SIZE && envelope[0] == VERSION) { "invalid key envelope" }
        val iv = envelope.copyOfRange(1, 1 + IV_SIZE)
        val ciphertext = envelope.copyOfRange(1 + IV_SIZE, envelope.size)
        val cipher = Cipher.getInstance("AES/GCM/NoPadding")
        cipher.init(Cipher.DECRYPT_MODE, key(), GCMParameterSpec(128, iv))
        return cipher.doFinal(ciphertext)
    }
}
