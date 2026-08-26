package network.sudharma.wallet

object TestnetChallengeConfig {
    const val DEFAULT_RPC_URL = "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com"
    const val INITIAL_GRANT_SUDH = "100"
    const val CHALLENGE_SEND_SUDH = "25"
    const val CHALLENGE_REWARD_SUDH = "50"
    const val MAX_ROUNDS = 5
    const val COOLDOWN_HOURS = 24

    // Set only after a dedicated challenge wallet is created and funded.
    // Keeping this null prevents the app from accidentally sending tester funds
    // to an unrelated address before the official wallet exists.
    val challengeDepositAddress: String? = null

    val faucetEnabled: Boolean = false
}
