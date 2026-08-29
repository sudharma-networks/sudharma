package network.sudharma.wallet.security

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class StrongBoxPolicyTest {
    @Test
    fun strongBoxIsPreferredOnlyWhenPlatformSupportsIt() {
        assertFalse(StrongBoxPolicy.shouldAttemptStrongBox(27))
        assertTrue(StrongBoxPolicy.shouldAttemptStrongBox(28))
        assertTrue(StrongBoxPolicy.shouldAttemptStrongBox(35))
    }
}
