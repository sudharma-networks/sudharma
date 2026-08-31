package network.sudharma.wallet.chain.sudharma

data class SudharmaTransaction(
    val id: String,
    val from: String,
    val to: String,
    val amount: Long,
    val fee: Long,
    val nonce: Long,
    val publicKey: ByteArray? = null,
    val signature: ByteArray? = null,
) {
    companion object {
        fun create(from: String, to: String, amount: Long, nonce: Long): SudharmaTransaction {
            require(from.isNotBlank()) { "sender cannot be blank" }
            require(to.isNotBlank()) { "receiver cannot be blank" }
            require(amount > 0L) { "amount must be positive" }
            require(nonce >= 0L) { "nonce cannot be negative" }
            val fee = calculateFee(amount)
            val canonical = "$from|$to|$amount|$fee|$nonce"
            val id = SudharmaCrypto.sha256(canonical.toByteArray()).toHex()
            return SudharmaTransaction(id, from, to, amount, fee, nonce)
        }

        fun calculateFee(amount: Long): Long {
            require(amount >= 0L) { "amount cannot be negative" }
            return Math.multiplyExact(amount, 10L) / 10_000L
        }

        private fun ByteArray.toHex(): String = joinToString("") { "%02x".format(it.toInt() and 0xff) }
    }

    fun signed(privateScalar: java.math.BigInteger): SudharmaTransaction {
        val key = SudharmaCrypto.keyFromPrivateScalar(privateScalar)
        require(SudharmaCrypto.addressFromPublicKey(key.publicKey) == from) { "key does not match sender" }
        return copy(
            publicKey = key.publicKey,
            signature = SudharmaCrypto.sign(privateScalar, id.toByteArray()),
        )
    }

    fun verify(): Boolean {
        val pub = publicKey ?: return false
        val sig = signature ?: return false
        if (SudharmaCrypto.addressFromPublicKey(pub) != from) return false
        val expected = create(from, to, amount, nonce)
        if (expected.id != id || expected.fee != fee) return false
        return SudharmaCrypto.verify(pub, id.toByteArray(), sig)
    }
}
