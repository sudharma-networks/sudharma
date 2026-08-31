package network.sudharma.wallet.security

data class PinAttemptState(
    val failedAttempts: Int = 0,
    val blockedUntilEpochMillis: Long = 0L,
)

object PinBackoffPolicy {
    private const val FREE_ATTEMPTS = 4
    private const val INITIAL_BACKOFF_MILLIS = 30_000L
    private const val MAX_BACKOFF_MILLIS = 15 * 60_000L

    fun canAttempt(state: PinAttemptState, nowEpochMillis: Long): Boolean =
        nowEpochMillis >= state.blockedUntilEpochMillis

    fun delayMillis(failedAttempts: Int): Long {
        if (failedAttempts <= FREE_ATTEMPTS) return 0L
        val exponent = (failedAttempts - FREE_ATTEMPTS - 1).coerceIn(0, 5)
        return (INITIAL_BACKOFF_MILLIS * (1L shl exponent)).coerceAtMost(MAX_BACKOFF_MILLIS)
    }

    fun recordFailure(state: PinAttemptState, nowEpochMillis: Long): PinAttemptState {
        val failedAttempts = if (state.failedAttempts == Int.MAX_VALUE) {
            Int.MAX_VALUE
        } else {
            state.failedAttempts + 1
        }
        val delay = delayMillis(failedAttempts)
        val blockedUntil = when {
            delay == 0L -> 0L
            nowEpochMillis > Long.MAX_VALUE - delay -> Long.MAX_VALUE
            else -> nowEpochMillis + delay
        }
        return PinAttemptState(failedAttempts, blockedUntil)
    }

    fun recordSuccess(@Suppress("UNUSED_PARAMETER") state: PinAttemptState): PinAttemptState =
        PinAttemptState()
}
