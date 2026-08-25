package network.sudharma.wallet

data class SplashPresentation(
    val animate: Boolean,
    val delayMillis: Long,
)

object SplashPresentationPolicy {
    fun forAnimatorScale(scale: Float): SplashPresentation = if (scale == 0f) {
        SplashPresentation(animate = false, delayMillis = 250L)
    } else {
        SplashPresentation(animate = true, delayMillis = 1_700L)
    }
}

object WalletPresentationPolicy {
    fun isSensitive(screen: WalletScreen): Boolean = screen in setOf(
        WalletScreen.RECOVERY,
        WalletScreen.CONFIRM_RECOVERY,
        WalletScreen.IMPORT,
        WalletScreen.BACKUP,
    )
}
