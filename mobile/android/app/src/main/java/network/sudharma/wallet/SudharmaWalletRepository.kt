package network.sudharma.wallet

import android.content.Context
import network.sudharma.wallet.chain.AssetBalance
import network.sudharma.wallet.chain.TransactionStatus
import network.sudharma.wallet.chain.sudharma.SudharmaChainAdapter
import network.sudharma.wallet.chain.sudharma.SudharmaRpcClient
import network.sudharma.wallet.recovery.RecoveryPhrase
import network.sudharma.wallet.recovery.SudharmaMobileDerivationV1
import network.sudharma.wallet.security.AppSecurityPreferences
import network.sudharma.wallet.security.WalletStore
import java.math.BigDecimal
import java.math.RoundingMode

class SudharmaWalletRepository(context: Context) {
    val walletStore = WalletStore(context)
    val security = AppSecurityPreferences(context)
    val preferences = WalletPreferences(context)

    data class LocalAccount(val address: String, val privateScalar: java.math.BigInteger)

    fun createNewWallet(): String = RecoveryPhrase.generate12()

    fun importWallet(phrase: String) {
        require(RecoveryPhrase.validate(phrase)) { "Invalid 12-word recovery phrase" }
        walletStore.saveRecoveryPhrase(phrase.trim().lowercase())
    }

    fun setPin(pin: String) = security.savePin(pin)
    fun verifyPin(pin: String): Boolean = security.verifyPin(pin)

    fun account(): LocalAccount {
        val phrase = walletStore.loadRecoveryPhrase()
        val derived = SudharmaMobileDerivationV1.derive(RecoveryPhrase.seed(phrase), 0)
        return LocalAccount(derived.address, derived.privateScalar)
    }

    suspend fun balance(): AssetBalance = adapter().balance(account().address)

    suspend fun send(to: String, amountText: String): TransactionStatus {
        val account = account()
        val adapter = adapter()
        require(adapter.validateAddress(to)) { "Invalid Sudharma address" }
        require(to != account.address) { "Cannot send to the same wallet" }
        val amount = parseCoinAmount(amountText)
        val remoteAccount = adapter.balance(account.address)
        val fee = adapter.estimateFee(amount).feeAtomic
        require(remoteAccount.amount.atomic >= Math.addExact(amount, fee)) { "Insufficient balance including fee" }
        val unsigned = adapter.unsigned(account.address, to, amount, remoteAccount.nextNonce)
        val signed = adapter.sign(unsigned, account.privateScalar)
        val status = adapter.submit(signed)
        preferences.addTransactionId(status.id)
        return status
    }

    suspend fun transactionStatuses(): List<TransactionStatus> {
        val adapter = adapter()
        return preferences.transactionIds().map { id ->
            runCatching { adapter.status(id) }.getOrElse {
                network.sudharma.wallet.chain.TransactionStatus(
                    id,
                    network.sudharma.wallet.chain.TransactionState.FAILED,
                )
            }
        }
    }

    fun resetWallet() {
        walletStore.clear()
        security.clear()
        preferences.clear()
    }

    private fun adapter(): SudharmaChainAdapter {
        val url = preferences.rpcUrl
        require(url.isNotBlank()) { "Sudharma Testnet RPC is not configured" }
        return SudharmaChainAdapter(SudharmaRpcClient(url))
    }

    companion object {
        fun parseCoinAmount(text: String): Long {
            val value = text.trim()
            require(value.matches(Regex("^(0|[1-9][0-9]*)(\\.[0-9]{1,8})?$"))) { "Invalid SUDH amount" }
            val decimal = BigDecimal(value).setScale(8, RoundingMode.UNNECESSARY)
            val atomic = decimal.movePointRight(8).longValueExact()
            require(atomic > 0) { "Amount must be greater than zero" }
            return atomic
        }
    }
}
