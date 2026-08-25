package network.sudharma.wallet.security

import android.content.Context
import network.sudharma.wallet.recovery.WalletMetadata
import java.security.SecureRandom
import java.util.Base64

class WalletStore(context: Context) {
    private val prefs = context.getSharedPreferences("sudharma_wallet_secure_v1", Context.MODE_PRIVATE)
    private val random = SecureRandom()

    fun hasWallet(): Boolean =
        prefs.contains(KEY_WRAPPED_DATA_KEY) &&
            prefs.contains(KEY_SECRET) &&
            prefs.contains(KEY_DERIVATION_PROFILE) &&
            prefs.contains(KEY_DERIVATION_VERSION) &&
            prefs.contains(KEY_ACCOUNT_INDEX)

    fun saveRecoveryPhrase(
        phrase: String,
        metadata: WalletMetadata = WalletMetadata.sudharmaMobileV1(accountIndex = 0),
    ) {
        require(phrase.isNotBlank())
        val dataKey = ByteArray(32).also(random::nextBytes)
        val wrappedDataKey = AndroidKeyStoreBox.wrap(dataKey)
        val encryptedSecret = WalletCipher.encrypt(dataKey, phrase.toByteArray(Charsets.UTF_8))
        prefs.edit()
            .putString(KEY_WRAPPED_DATA_KEY, Base64.getEncoder().encodeToString(wrappedDataKey))
            .putString(KEY_SECRET, Base64.getEncoder().encodeToString(encryptedSecret))
            .putString(KEY_DERIVATION_PROFILE, metadata.derivationProfile)
            .putInt(KEY_DERIVATION_VERSION, metadata.derivationVersion)
            .putInt(KEY_ACCOUNT_INDEX, metadata.accountIndex)
            .apply()
        dataKey.fill(0)
    }

    fun loadRecoveryPhrase(): String {
        val wrapped = decodePref(KEY_WRAPPED_DATA_KEY)
        val encrypted = decodePref(KEY_SECRET)
        val dataKey = AndroidKeyStoreBox.unwrap(wrapped)
        return try {
            WalletCipher.decrypt(dataKey, encrypted).toString(Charsets.UTF_8)
        } finally {
            dataKey.fill(0)
        }
    }

    fun loadMetadata(): WalletMetadata {
        require(hasWallet()) { "wallet metadata is missing" }
        return WalletMetadata.restore(
            derivationProfile = requireNotNull(prefs.getString(KEY_DERIVATION_PROFILE, null)),
            derivationVersion = prefs.getInt(KEY_DERIVATION_VERSION, 0),
            accountIndex = prefs.getInt(KEY_ACCOUNT_INDEX, -1),
        )
    }

    fun clear() {
        prefs.edit().clear().apply()
    }

    private fun decodePref(key: String): ByteArray {
        val value = requireNotNull(prefs.getString(key, null)) { "wallet is not initialized" }
        return Base64.getDecoder().decode(value)
    }

    companion object {
        private const val KEY_WRAPPED_DATA_KEY = "wrapped_data_key"
        private const val KEY_SECRET = "encrypted_recovery"
        private const val KEY_DERIVATION_PROFILE = "derivation_profile"
        private const val KEY_DERIVATION_VERSION = "derivation_version"
        private const val KEY_ACCOUNT_INDEX = "account_index"
    }
}
