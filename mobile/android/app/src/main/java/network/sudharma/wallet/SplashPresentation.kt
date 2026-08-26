package network.sudharma.wallet

data class SplashPresentation(
    val animate: Boolean,
    val delayMillis: Long,
    val title: String,
    val subtitle: String,
    val haloPulseMillis: Long,
)

object SplashPresentationPolicy {
    fun forAnimatorScale(animatorScale: Float): SplashPresentation {
        val animate = animatorScale > 0f
        return SplashPresentation(
            animate = animate,
            delayMillis = if (animate) 1_850L else 250L,
            title = "SUDHARMA",
            subtitle = "TESTNET WALLET",
            haloPulseMillis = 900L,
        )
    }
}
