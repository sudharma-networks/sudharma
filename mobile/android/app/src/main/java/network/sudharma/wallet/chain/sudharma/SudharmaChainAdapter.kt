package network.sudharma.wallet.chain.sudharma

import network.sudharma.wallet.chain.AssetAmount
import network.sudharma.wallet.chain.AssetBalance
import network.sudharma.wallet.chain.ChainAdapter
import network.sudharma.wallet.chain.FeeQuote
import network.sudharma.wallet.chain.NetworkId
import network.sudharma.wallet.chain.SignedTransfer
import network.sudharma.wallet.chain.TransactionState
import network.sudharma.wallet.chain.TransactionStatus
import network.sudharma.wallet.chain.UnsignedTransfer
import java.math.BigInteger
import java.util.Base64

class SudharmaChainAdapter(
    private val rpc: SudharmaRpcClient,
) : ChainAdapter {
    override val network = NetworkId("sudharma", "testnet", isTestnet = true)
    override val symbol: String = "SUDH"

    override fun validateAddress(address: String): Boolean = address.matches(Regex("^[0-9a-f]{40}$"))

    override suspend fun balance(address: String): AssetBalance {
        require(validateAddress(address)) { "invalid Sudharma address" }
        val account = rpc.account(address)
        return AssetBalance(
            network = network,
            address = account.address,
            amount = AssetAmount(symbol, account.balance, 8),
            confirmedNonce = account.confirmedNonce,
            nextNonce = account.nextNonce,
        )
    }

    override fun estimateFee(amountAtomic: Long): FeeQuote =
        FeeQuote(amountAtomic, SudharmaTransaction.calculateFee(amountAtomic))

    fun unsigned(from: String, to: String, amountAtomic: Long, nonce: Long): UnsignedTransfer {
        require(validateAddress(from)) { "invalid sender address" }
        require(validateAddress(to)) { "invalid receiver address" }
        require(amountAtomic > 0) { "amount must be positive" }
        val fee = estimateFee(amountAtomic).feeAtomic
        return UnsignedTransfer(network, from, to, amountAtomic, fee, nonce)
    }

    fun sign(unsigned: UnsignedTransfer, privateScalar: BigInteger): SignedTransfer {
        require(unsigned.network == network) { "wrong network" }
        val tx = SudharmaTransaction.create(
            from = unsigned.from,
            to = unsigned.to,
            amount = unsigned.amountAtomic,
            nonce = unsigned.nonce,
        ).signed(privateScalar)
        require(tx.fee == unsigned.feeAtomic) { "fee changed" }
        return SignedTransfer(tx.id, encode(tx))
    }

    override suspend fun submit(transfer: SignedTransfer): TransactionStatus {
        val tx = decode(transfer.payload)
        require(tx.id == transfer.transactionId && tx.verify()) { "invalid signed transfer" }
        val result = rpc.submit(tx)
        if (!result.accepted) return TransactionStatus(tx.id, TransactionState.FAILED)
        return TransactionStatus(tx.id, TransactionState.PENDING)
    }

    override suspend fun status(transactionId: String): TransactionStatus = try {
        val remote = rpc.transaction(transactionId)
        TransactionStatus(
            id = transactionId,
            state = when (remote.status.lowercase()) {
                "confirmed" -> TransactionState.CONFIRMED
                "pending" -> TransactionState.PENDING
                else -> TransactionState.FAILED
            },
            confirmations = remote.confirmations,
            blockHeight = remote.blockHeight,
        )
    } catch (e: SudharmaRpcClient.RpcException) {
        if (e.statusCode == 404) TransactionStatus(transactionId, TransactionState.NOT_FOUND) else throw e
    }

    private fun encode(tx: SudharmaTransaction): ByteArray {
        val pub = Base64.getEncoder().encodeToString(requireNotNull(tx.publicKey))
        val sig = Base64.getEncoder().encodeToString(requireNotNull(tx.signature))
        return listOf(tx.id, tx.from, tx.to, tx.amount, tx.fee, tx.nonce, pub, sig)
            .joinToString("|")
            .toByteArray()
    }

    private fun decode(payload: ByteArray): SudharmaTransaction {
        val p = payload.toString(Charsets.UTF_8).split('|')
        require(p.size == 8) { "invalid signed payload" }
        return SudharmaTransaction(
            id = p[0],
            from = p[1],
            to = p[2],
            amount = p[3].toLong(),
            fee = p[4].toLong(),
            nonce = p[5].toLong(),
            publicKey = Base64.getDecoder().decode(p[6]),
            signature = Base64.getDecoder().decode(p[7]),
        )
    }
}
