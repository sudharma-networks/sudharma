package network.sudharma.wallet.backup

import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CloudBackupBoundaryTest {
    @Test
    fun missingGoogleConfigurationDoesNotBlockCoreWallet() {
        val boundary = CloudBackupBoundary(UnconfiguredGoogleBackup)

        assertFalse(boundary.available)
    }

    @Test
    fun providerReceivesOnlyLocallyEncryptedCiphertext() = runBlocking {
        val provider = RecordingBackupProvider()
        val boundary = CloudBackupBoundary(provider)
        val phrase = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

        boundary.uploadRecovery(phrase.toByteArray(), "a-strong-backup-password")

        val uploaded = requireNotNull(provider.uploaded)
        assertFalse(uploaded.toString(Charsets.UTF_8).contains("abandon"))
        assertTrue(
            phrase.toByteArray().contentEquals(
                EncryptedBackup.decrypt(uploaded, "a-strong-backup-password"),
            ),
        )
    }

    private class RecordingBackupProvider : CloudBackupProvider {
        override val configured: Boolean = true
        var uploaded: ByteArray? = null

        override suspend fun upload(ciphertext: ByteArray) {
            uploaded = ciphertext.copyOf()
        }

        override suspend fun download(): ByteArray? = uploaded?.copyOf()
    }
}
