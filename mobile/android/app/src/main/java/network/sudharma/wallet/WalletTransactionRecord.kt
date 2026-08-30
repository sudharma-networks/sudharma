package network.sudharma.wallet

enum class TransactionDirection {
    SENT,
    RECEIVED,
    ;

    companion object {
        fun parse(raw: String): TransactionDirection? = entries.firstOrNull {
            it.name.equals(raw, ignoreCase = true)
        }
    }
}

data class WalletTransactionRecord(
    val id: String,
    val direction: TransactionDirection,
    val amountAtomic: Long,
    val counterparty: String,
    val feeAtomic: Long = 0L,
    val timestampMs: Long = System.currentTimeMillis(),
) {
    init {
        require(id.matches(Regex("^[0-9a-f]{64}$"))) { "invalid transaction id" }
        require(amountAtomic > 0L) { "amount must be positive" }
        require(counterparty.matches(Regex("^[0-9a-f]{40}$"))) { "invalid counterparty" }
        require(feeAtomic >= 0L) { "fee cannot be negative" }
    }

    fun encode(): String = listOf(
        id,
        direction.name,
        amountAtomic.toString(),
        counterparty,
        feeAtomic.toString(),
        timestampMs.toString(),
    ).joinToString("|")

    companion object {
        private val pattern = Regex(
            "^([0-9a-f]{64})\\|(SENT|RECEIVED)\\|(\\d+)\\|([0-9a-f]{40})\\|(\\d+)\\|(\\d+)$",
        )

        fun decode(raw: String): WalletTransactionRecord? {
            val match = pattern.matchEntire(raw.trim()) ?: return null
            return WalletTransactionRecord(
                id = match.groupValues[1],
                direction = TransactionDirection.valueOf(match.groupValues[2]),
                amountAtomic = match.groupValues[3].toLongOrNull() ?: return null,
                counterparty = match.groupValues[4],
                feeAtomic = match.groupValues[5].toLongOrNull() ?: return null,
                timestampMs = match.groupValues[6].toLongOrNull() ?: return null,
            )
        }
    }
}

data class WalletActivityItem(
    val record: WalletTransactionRecord,
    val state: network.sudharma.wallet.chain.TransactionState,
    val confirmations: Long = 0,
)
