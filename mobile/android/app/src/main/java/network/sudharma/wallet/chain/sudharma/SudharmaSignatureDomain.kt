package network.sudharma.wallet.chain.sudharma

object SudharmaSignatureDomain {
    const val LEGACY = 1
    const val NETWORK_BOUND = 2

    const val DEFAULT_NETWORK = "sudharma-testnet-1"
    const val MAINNET_NETWORK = "sudharma-mainnet-1"

    fun signingMessage(domain: Int, network: String, txId: String): ByteArray = when (domain) {
        LEGACY -> txId.toByteArray()
        NETWORK_BOUND -> "sudharma-tx-v2|$network|$txId".toByteArray()
        else -> throw IllegalArgumentException("unknown signature domain: $domain")
    }
}
