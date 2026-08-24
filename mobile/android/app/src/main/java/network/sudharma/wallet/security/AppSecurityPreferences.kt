package network.sudharma.wallet.security

import android.content.Context

class AppSecurityPreferences(context: Context) {
    private val prefs = context.getSharedPreferences("sudharma_wallet_security_v1", Context.MODE_PRIVATE)

    fun savePin(pin: String) {
        val record = PinVerifier.create(pin)
        prefs.edit()
            .putInt("pin_version", record.version)
            .putInt("pin_iterations", record.iterations)
            .putString("pin_salt", record.saltHex)
            .putString("pin_hash", record.hashHex)
            .apply()
    }

    fun hasPin(): Boolean = prefs.contains("pin_hash")

    fun verifyPin(pin: String): Boolean {
        if (!hasPin()) return false
        val record = PinVerifier.PinRecord(
            version = prefs.getInt("pin_version", 1),
            iterations = prefs.getInt("pin_iterations", 0),
            saltHex = prefs.getString("pin_salt", "") ?: "",
            hashHex = prefs.getString("pin_hash", "") ?: "",
        )
        return PinVerifier.verify(pin, record)
    }

    var biometricEnabled: Boolean
        get() = prefs.getBoolean("biometric_enabled", false)
        set(value) { prefs.edit().putBoolean("biometric_enabled", value).apply() }

    fun clear() = prefs.edit().clear().apply()
}
