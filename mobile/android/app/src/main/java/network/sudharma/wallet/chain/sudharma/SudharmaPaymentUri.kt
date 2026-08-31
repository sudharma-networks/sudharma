package network.sudharma.wallet.chain.sudharma

object SudharmaPaymentUri {
    data class Payment(
        val address: String,
        val network: String = "testnet",
        val version: Int = 1,
        val amountAtomic: Long? = null,
    )

    fun encode(address: String, amountAtomic: Long? = null): String {
        require(address.matches(Regex("^[0-9a-f]{40}$"))) { "invalid address" }
        require(amountAtomic == null || amountAtomic > 0) { "amount must be positive" }
        val amount = amountAtomic?.let { "&amount=$it" }.orEmpty()
        return "sudharma:$address?network=testnet&v=1$amount"
    }

    fun decode(value: String): Payment? {
        val trimmed = value.trim()
        if (trimmed.matches(Regex("^[0-9a-f]{40}$"))) return Payment(trimmed)
        val match = Regex(
            "^sudharma:([0-9a-f]{40})\\?network=testnet&v=1(?:&amount=([1-9][0-9]*))?$",
        ).matchEntire(trimmed)
            ?: return null
        val amount = match.groupValues[2].takeIf(String::isNotEmpty)?.toLongOrNull() ?: run {
            if (match.groupValues[2].isNotEmpty()) return null
            null
        }
        return Payment(address = match.groupValues[1], amountAtomic = amount)
    }

    fun parse(value: String): String? = decode(value)?.address
}
