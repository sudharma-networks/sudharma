package network.sudharma.wallet

object RpcEndpointPolicy {
    fun resolve(storedValue: String?, allowDebugOverride: Boolean): String {
        val normalized = storedValue?.trim()?.trimEnd('/').orEmpty()
        return if (allowDebugOverride && normalized.isNotEmpty()) {
            normalized
        } else {
            TestnetChallengeConfig.DEFAULT_RPC_URL
        }
    }
}
