package network.sudharma.wallet

object TestnetChallengePolicy {
    fun matchesOfficialChallenge(
        info: TestnetFaucetClient.Info?,
        to: String,
        amountAtomic: Long,
        coinAtomic: Long = 100_000_000L,
    ): Boolean {
        if (info?.enabled != true) return false
        val challengeAmount = info.challengeSendSudh.toLongOrNull()
            ?.let { runCatching { Math.multiplyExact(it, coinAtomic) }.getOrNull() }
            ?: return false
        return to == info.challengeAddress && amountAtomic == challengeAmount
    }
}
