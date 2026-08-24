package network.sudharma.wallet.chain.sudharma

object SudharmaPaymentUri {
    fun encode(address: String): String {
        require(address.matches(Regex("^[0-9a-f]{40}$"))) { "invalid address" }
        return "sudharma:$address?network=testnet&v=1"
    }

    fun parse(value: String): String? {
        val trimmed = value.trim()
        if (trimmed.matches(Regex("^[0-9a-f]{40}$"))) return trimmed
        val match = Regex("^sudharma:([0-9a-f]{40})\\?network=testnet&v=1$").matchEntire(trimmed)
            ?: return null
        return match.groupValues[1]
    }
}
