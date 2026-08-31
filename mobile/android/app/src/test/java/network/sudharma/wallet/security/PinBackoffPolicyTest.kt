package network.sudharma.wallet.security

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PinBackoffPolicyTest {
    @Test
    fun fifthFailedAttemptStartsThirtySecondBackoff() {
        var state = PinAttemptState()
        repeat(5) { state = PinBackoffPolicy.recordFailure(state, nowEpochMillis = 100_000L) }

        assertEquals(5, state.failedAttempts)
        assertEquals(130_000L, state.blockedUntilEpochMillis)
        assertFalse(PinBackoffPolicy.canAttempt(state, nowEpochMillis = 129_999L))
        assertTrue(PinBackoffPolicy.canAttempt(state, nowEpochMillis = 130_000L))
    }

    @Test
    fun repeatedFailuresDoubleDelayAndCapAtFifteenMinutes() {
        assertEquals(30_000L, PinBackoffPolicy.delayMillis(failedAttempts = 5))
        assertEquals(60_000L, PinBackoffPolicy.delayMillis(failedAttempts = 6))
        assertEquals(900_000L, PinBackoffPolicy.delayMillis(failedAttempts = 20))
    }

    @Test
    fun successfulAuthenticationClearsFailureState() {
        val blocked = PinAttemptState(failedAttempts = 9, blockedUntilEpochMillis = 999_999L)
        assertEquals(PinAttemptState(), PinBackoffPolicy.recordSuccess(blocked))
    }
}
