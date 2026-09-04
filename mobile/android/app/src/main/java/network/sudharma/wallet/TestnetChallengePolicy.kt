package network.sudharma.wallet

object TestnetChallengePolicy {
    fun matchesOfficialChallenge(
        info: TestnetFaucetClient.Info?,
        to: String,
        amountAtomic: Long,
        coinAtomic: Long = 100_000_000L,
    ): Boolean {
        if (info?.enabled != true) return false
        val challengeAmount = runCatching {
            Math.multiplyExact(info.challengeSendSudh.toLong(), coinAtomic)
        }.getOrNull() ?: return false
        return to == info.challengeAddress && amountAtomic == challengeAmount
    }
}
