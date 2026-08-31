package network.sudharma.wallet.recovery

class WalletMetadata private constructor(
    val derivationProfile: String,
    val derivationVersion: Int,
    val accountIndex: Int,
) {
    val profileId: String = "$derivationProfile-v$derivationVersion"

    companion object {
        private const val SUDHARMA_MOBILE = "sudharma-mobile"
        private const val SUDHARMA_MOBILE_VERSION = 1

        fun sudharmaMobileV1(accountIndex: Int): WalletMetadata = restore(
            derivationProfile = SUDHARMA_MOBILE,
            derivationVersion = SUDHARMA_MOBILE_VERSION,
            accountIndex = accountIndex,
        )

        fun restore(
            derivationProfile: String,
            derivationVersion: Int,
            accountIndex: Int,
        ): WalletMetadata {
            require(accountIndex >= 0) { "account index cannot be negative" }
            require(
                derivationProfile == SUDHARMA_MOBILE &&
                    derivationVersion == SUDHARMA_MOBILE_VERSION,
            ) { "unsupported wallet derivation profile" }
            return WalletMetadata(derivationProfile, derivationVersion, accountIndex)
        }
    }
}
