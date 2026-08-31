package network.sudharma.wallet.backup

class CloudBackupBoundary(
    private val provider: CloudBackupProvider,
) {
    val available: Boolean
        get() = provider.configured

    suspend fun uploadRecovery(recoveryBytes: ByteArray, backupPassword: String) {
        check(provider.configured) { "Cloud backup is not configured" }
        val ciphertext = EncryptedBackup.encrypt(recoveryBytes, backupPassword)
        provider.upload(ciphertext)
    }
}
