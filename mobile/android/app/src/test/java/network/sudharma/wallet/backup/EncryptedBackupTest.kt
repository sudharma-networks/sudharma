package network.sudharma.wallet.backup

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class EncryptedBackupTest {
    @Test fun backupEnvelopeContainsNoPlainRecoveryWords() {
        val phrase = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
        val envelope = EncryptedBackup.encrypt(phrase.toByteArray(), "a-strong-backup-password")
        assertFalse(String(envelope).contains("abandon"))
        assertTrue(phrase.toByteArray().contentEquals(EncryptedBackup.decrypt(envelope, "a-strong-backup-password")))
    }

    @Test(expected = IllegalArgumentException::class)
    fun shortBackupPasswordRejected() {
        EncryptedBackup.encrypt("secret".toByteArray(), "short")
    }
}
