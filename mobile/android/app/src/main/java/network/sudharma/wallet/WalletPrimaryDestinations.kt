package network.sudharma.wallet

/** Stable home destinations that must survive every later wallet release. */
object WalletPrimaryDestinations {
    val activityAndHistory: List<WalletScreen> = listOf(
        WalletScreen.ACTIVITY,
        WalletScreen.HISTORY,
    )
}
