package network.sudharma.wallet

object ExplorerLinks {
    const val PUBLIC_EXPLORER_BASE_URL =
        "https://feature-website-foundation.d2mqyt0bt8sl9s.amplifyapp.com"

    fun transactionPath(transactionId: String): String =
        "/explorer/tx?id=$transactionId"

    fun transactionUrl(transactionId: String): String =
        PUBLIC_EXPLORER_BASE_URL + transactionPath(transactionId)
}
