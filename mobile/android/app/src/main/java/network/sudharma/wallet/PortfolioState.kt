package network.sudharma.wallet

data class ActivitySummary(
    val id: String,
    val status: String,
)

data class PortfolioState(
    val loading: Boolean,
    val balanceAtomic: Long?,
    val message: String?,
    val activity: List<ActivitySummary>,
    val networkBadge: String = "TESTNET",
    val assetSymbols: List<String> = listOf("SUDH"),
    val mainnetEnabled: Boolean = false,
    val swapEnabled: Boolean = false,
    val swapLabel: String = "Coming later",
) {
    companion object {
        fun loading(): PortfolioState = PortfolioState(
            loading = true,
            balanceAtomic = null,
            message = null,
            activity = emptyList(),
        )

        fun offline(message: String): PortfolioState = PortfolioState(
            loading = false,
            balanceAtomic = null,
            message = message,
            activity = emptyList(),
        )

        fun loaded(
            balanceAtomic: Long,
            activity: List<ActivitySummary>,
        ): PortfolioState = PortfolioState(
            loading = false,
            balanceAtomic = balanceAtomic,
            message = null,
            activity = activity,
        )
    }
}
