package network.sudharma.wallet.chain

import java.math.BigDecimal
import java.math.RoundingMode

data class NetworkId(
    val chain: String,
    val network: String,
    val isTestnet: Boolean,
) {
    init {
        require(chain.isNotBlank()) { "chain cannot be blank" }
        require(network.isNotBlank()) { "network cannot be blank" }
    }

    val key: String = "$chain:$network"
}

data class AssetAmount(
    val symbol: String,
    val atomic: Long,
    val decimals: Int,
) {
    init {
        require(symbol.isNotBlank()) { "symbol cannot be blank" }
        require(atomic >= 0L) { "atomic amount cannot be negative" }
        require(decimals in 0..18) { "invalid decimals" }
    }

    fun formatted(): String = BigDecimal.valueOf(atomic)
        .movePointLeft(decimals)
        .setScale(decimals, RoundingMode.UNNECESSARY)
        .toPlainString()
}

data class AssetBalance(
    val network: NetworkId,
    val address: String,
    val amount: AssetAmount,
    val confirmedNonce: Long,
    val nextNonce: Long,
)

data class FeeQuote(
    val amountAtomic: Long,
    val feeAtomic: Long,
)

data class UnsignedTransfer(
    val network: NetworkId,
    val from: String,
    val to: String,
    val amountAtomic: Long,
    val feeAtomic: Long,
    val nonce: Long,
)

data class SignedTransfer(
    val transactionId: String,
    val payload: ByteArray,
) {
    override fun equals(other: Any?): Boolean = other is SignedTransfer &&
        transactionId == other.transactionId && payload.contentEquals(other.payload)

    override fun hashCode(): Int = 31 * transactionId.hashCode() + payload.contentHashCode()
}

enum class TransactionState { PENDING, CONFIRMED, FAILED, NOT_FOUND }

data class TransactionStatus(
    val id: String,
    val state: TransactionState,
    val confirmations: Long = 0,
    val blockHeight: Long? = null,
)

interface ChainAdapter {
    val network: NetworkId
    val symbol: String
    fun validateAddress(address: String): Boolean
    suspend fun balance(address: String): AssetBalance
    fun estimateFee(amountAtomic: Long): FeeQuote
    suspend fun submit(transfer: SignedTransfer): TransactionStatus
    suspend fun status(transactionId: String): TransactionStatus
}
