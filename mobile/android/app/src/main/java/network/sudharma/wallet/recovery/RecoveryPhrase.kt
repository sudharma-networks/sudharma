package network.sudharma.wallet.recovery

import org.web3j.crypto.MnemonicUtils
import java.security.SecureRandom

object RecoveryPhrase {
    private val random = SecureRandom()

    fun generate12(): String {
        val entropy = ByteArray(16)
        random.nextBytes(entropy)
        return MnemonicUtils.generateMnemonic(entropy)
    }

    fun validate(phrase: String): Boolean = runCatching {
        MnemonicUtils.validateMnemonic(phrase.trim().lowercase())
    }.getOrDefault(false)

    fun seed(phrase: String): ByteArray {
        val normalized = phrase.trim().lowercase()
        require(validate(normalized)) { "invalid recovery phrase" }
        return MnemonicUtils.generateSeed(normalized, "")
    }
}
