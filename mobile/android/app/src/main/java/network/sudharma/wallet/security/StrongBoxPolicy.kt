package network.sudharma.wallet.security

internal object StrongBoxPolicy {
    private const val STRONGBOX_API_LEVEL = 28

    fun shouldAttemptStrongBox(sdkInt: Int): Boolean = sdkInt >= STRONGBOX_API_LEVEL
}
