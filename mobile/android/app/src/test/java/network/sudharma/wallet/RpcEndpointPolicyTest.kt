package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Test

class RpcEndpointPolicyTest {
    private val official = TestnetChallengeConfig.DEFAULT_RPC_URL

    @Test
    fun blankLegacyPreferenceMigratesToOfficialEndpoint() {
        assertEquals(official, RpcEndpointPolicy.resolve("   ", allowDebugOverride = true))
    }

    @Test
    fun missingPreferenceUsesOfficialEndpoint() {
        assertEquals(official, RpcEndpointPolicy.resolve(null, allowDebugOverride = true))
    }

    @Test
    fun productionBuildAlwaysUsesOfficialEndpoint() {
        assertEquals(
            official,
            RpcEndpointPolicy.resolve("https://example.invalid", allowDebugOverride = false),
        )
    }

    @Test
    fun debugBuildMayUseExplicitOverride() {
        assertEquals(
            "https://debug.example",
            RpcEndpointPolicy.resolve(" https://debug.example/ ", allowDebugOverride = true),
        )
    }
}
