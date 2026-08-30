package network.sudharma.wallet

class TestnetAutomationCoordinator(
    private val walletReady: () -> Boolean,
    private val faucetEnabled: suspend () -> Boolean,
    private val balanceAtomic: suspend () -> Long?,
    private val requestInitial: suspend () -> Unit,
    private val pendingChallengeId: () -> String?,
    private val transactionConfirmed: suspend (String) -> Boolean,
    private val transactionFailed: suspend (String) -> Boolean = { false },
    private val claimChallenge: suspend (String) -> Unit,
    private val clearPendingChallenge: () -> Unit,
    private val now: () -> Long = System::currentTimeMillis,
    private val initialRetryMillis: Long = 30_000L,
) {
    private var lastInitialAttemptAt: Long? = null

    suspend fun tick() {
        if (!walletReady()) return
        if (!runCatching { faucetEnabled() }.getOrDefault(false)) return

        val balance = runCatching { balanceAtomic() }.getOrNull()
        if (balance != null) {
            val currentTime = now()
            val attemptedRecently = lastInitialAttemptAt?.let { currentTime - it < initialRetryMillis } == true
            if (TestnetAutomationPolicy.shouldRequestInitialGrant(
                    faucetEnabled = true,
                    balanceAtomic = balance,
                    requestAlreadyAttempted = attemptedRecently,
                )
            ) {
                lastInitialAttemptAt = currentTime
                runCatching { requestInitial() }
            }
            if (balance > 0L) lastInitialAttemptAt = null
        }

        val transactionId = pendingChallengeId() ?: return
        val confirmed = runCatching { transactionConfirmed(transactionId) }.getOrDefault(false)
        if (!confirmed) {
            val failed = runCatching { transactionFailed(transactionId) }.getOrDefault(false)
            if (failed) clearPendingChallenge()
            return
        }
        if (!TestnetAutomationPolicy.shouldClaimChallengeReward(
                challengeMode = true,
                transactionConfirmed = true,
                claimAlreadyAttempted = false,
            )
        ) return

        runCatching { claimChallenge(transactionId) }
            .onSuccess { clearPendingChallenge() }
    }
}
