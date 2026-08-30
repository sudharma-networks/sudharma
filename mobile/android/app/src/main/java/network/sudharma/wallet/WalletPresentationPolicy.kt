package network.sudharma.wallet

data class SplashPresentation(
    val animate: Boolean,
    val delayMillis: Long,
    val title: String,
    val subtitle: String,
    val haloPulseMillis: Long,
)

object SplashPresentationPolicy {
    fun forAnimatorScale(scale: Float): SplashPresentation = if (scale == 0f) {
        SplashPresentation(
            animate = false,
            delayMillis = 250L,
            title = "SUDHARMA",
            subtitle = "TESTNET WALLET",
            haloPulseMillis = 900L,
        )
    } else {
        SplashPresentation(
            animate = true,
            delayMillis = 1_850L,
            title = "SUDHARMA",
            subtitle = "TESTNET WALLET",
            haloPulseMillis = 900L,
        )
    }
}

data class TestnetAutomationPresentation(
    val showManualInitialRequest: Boolean,
    val showManualChallengeClaim: Boolean,
    val initialFundingMessage: String,
    val challengeRewardMessage: String,
    val homeRefreshMillis: Long,
)

object TestnetAutomationPresentationPolicy {
    fun automatic(): TestnetAutomationPresentation = TestnetAutomationPresentation(
        showManualInitialRequest = false,
        showManualChallengeClaim = false,
        initialFundingMessage = "Eligible zero-balance wallets request Test SUDH automatically. The wallet keeps retrying safely and updates the balance after the network sees the payout.",
        challengeRewardMessage = "After you authorize the official challenge payment, the wallet automatically waits for confirmation and claims the eligible Test SUDH reward.",
        homeRefreshMillis = 5_000L,
    )
}

object WalletPresentationPolicy {
    fun isSensitive(screen: WalletScreen): Boolean = screen in setOf(
        WalletScreen.RECOVERY,
        WalletScreen.CONFIRM_RECOVERY,
        WalletScreen.IMPORT,
        WalletScreen.BACKUP,
    )
}
