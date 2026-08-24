package network.sudharma.wallet.recovery

import network.sudharma.wallet.chain.sudharma.SudharmaCrypto
import java.math.BigInteger
import java.nio.ByteBuffer
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec

object SudharmaMobileDerivationV1 {
    private const val PROFILE = "sudharma-mobile-v1"
    private val curveOrder = BigInteger(
        "FFFFFFFF00000000FFFFFFFFFFFFFFFFBCE6FAADA7179E84F3B9CAC2FC632551",
        16,
    )
    private val domain = "sudharma-mobile-derivation-v1".toByteArray()

    data class DerivedAccount(
        val profile: String,
        val accountIndex: Int,
        val privateScalar: BigInteger,
        val publicKey: ByteArray,
        val address: String,
    )

    fun derive(seed: ByteArray, accountIndex: Int): DerivedAccount {
        require(seed.isNotEmpty()) { "seed cannot be empty" }
        require(accountIndex >= 0) { "account index cannot be negative" }

        val mac = Mac.getInstance("HmacSHA256")
        mac.init(SecretKeySpec(seed, "HmacSHA256"))

        var counter = 0
        while (true) {
            val suffix = ByteBuffer.allocate(8)
                .putInt(accountIndex)
                .putInt(counter)
                .array()
            val candidate = BigInteger(1, mac.doFinal(domain + suffix))
            if (candidate.signum() > 0 && candidate < curveOrder) {
                val key = SudharmaCrypto.keyFromPrivateScalar(candidate)
                return DerivedAccount(
                    profile = PROFILE,
                    accountIndex = accountIndex,
                    privateScalar = candidate,
                    publicKey = key.publicKey,
                    address = SudharmaCrypto.addressFromPublicKey(key.publicKey),
                )
            }
            counter++
            check(counter != Int.MAX_VALUE) { "unable to derive valid private scalar" }
        }
    }
}
