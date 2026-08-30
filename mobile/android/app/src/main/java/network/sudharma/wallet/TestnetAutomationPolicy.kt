package network.sudharma.wallet

object TestnetAutomationPolicy {
    fun shouldRequestInitialGrant(
        faucetEnabled: Boolean,
        balanceAtomic: Long,
        requestAlreadyAttempted: Boolean,
    ): Boolean = faucetEnabled && balanceAtomic == 0L && !requestAlreadyAttempted

    fun shouldClaimChallengeReward(
        challengeMode: Boolean,
        transactionConfirmed: Boolean,
        claimAlreadyAttempted: Boolean,
    ): Boolean = challengeMode && transactionConfirmed && !claimAlreadyAttempted
}
