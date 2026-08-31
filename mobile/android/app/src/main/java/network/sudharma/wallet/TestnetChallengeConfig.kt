package network.sudharma.wallet

object TestnetChallengeConfig {
    const val DEFAULT_RPC_URL = "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com"
    const val INITIAL_GRANT_SUDH = "100"
    const val CHALLENGE_SEND_SUDH = "25"
    const val CHALLENGE_REWARD_SUDH = "50"
    const val MAX_ROUNDS = 5
    const val COOLDOWN_HOURS = 24

    // Legacy-safe defaults. The working wallet obtains the live challenge
    // address and enablement state from GET /v1/faucet/info so key rotation
    // does not require publishing a new APK.
    val challengeDepositAddress: String? = null
    val faucetEnabled: Boolean = false
}
