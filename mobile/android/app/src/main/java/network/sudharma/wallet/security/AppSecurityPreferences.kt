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
            .putInt(KEY_FAILED_ATTEMPTS, 0)
            .putLong(KEY_BLOCKED_UNTIL, 0L)
            .apply()
    }

    fun hasPin(): Boolean = prefs.contains("pin_hash")

    fun verifyPin(pin: String, nowEpochMillis: Long = System.currentTimeMillis()): Boolean {
        if (!hasPin()) return false
        val attemptState = loadAttemptState()
        if (!PinBackoffPolicy.canAttempt(attemptState, nowEpochMillis)) return false
        val record = PinVerifier.PinRecord(
            version = prefs.getInt("pin_version", 1),
            iterations = prefs.getInt("pin_iterations", 0),
            saltHex = prefs.getString("pin_salt", "") ?: "",
            hashHex = prefs.getString("pin_hash", "") ?: "",
        )
        val verified = PinVerifier.verify(pin, record)
        saveAttemptState(
            if (verified) {
                PinBackoffPolicy.recordSuccess(attemptState)
            } else {
                PinBackoffPolicy.recordFailure(attemptState, nowEpochMillis)
            },
        )
        return verified
    }

    var biometricEnabled: Boolean
        get() = prefs.getBoolean("biometric_enabled", false)
        set(value) { prefs.edit().putBoolean("biometric_enabled", value).apply() }

    fun clear() = prefs.edit().clear().apply()

    private fun loadAttemptState(): PinAttemptState = PinAttemptState(
        failedAttempts = prefs.getInt(KEY_FAILED_ATTEMPTS, 0),
        blockedUntilEpochMillis = prefs.getLong(KEY_BLOCKED_UNTIL, 0L),
    )

    private fun saveAttemptState(state: PinAttemptState) {
        prefs.edit()
            .putInt(KEY_FAILED_ATTEMPTS, state.failedAttempts)
            .putLong(KEY_BLOCKED_UNTIL, state.blockedUntilEpochMillis)
            .apply()
    }

    companion object {
        private const val KEY_FAILED_ATTEMPTS = "pin_failed_attempts"
        private const val KEY_BLOCKED_UNTIL = "pin_blocked_until_epoch_millis"
    }
}
