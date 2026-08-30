package network.sudharma.wallet

import android.content.Context
import network.sudharma.wallet.chain.AssetBalance
import network.sudharma.wallet.chain.TransactionStatus
import network.sudharma.wallet.chain.sudharma.SudharmaChainAdapter
import network.sudharma.wallet.chain.sudharma.SudharmaRpcClient
import network.sudharma.wallet.recovery.RecoveryPhrase
import network.sudharma.wallet.recovery.SudharmaMobileDerivationV1
import network.sudharma.wallet.security.AppSecurityPreferences
import network.sudharma.wallet.security.LegacyEncryptedWalletImporter
import network.sudharma.wallet.security.WalletStore
import network.sudharma.wallet.chain.sudharma.SudharmaCrypto
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

    fun importTreasuryWallet(jsonBytes: ByteArray, password: CharArray) {
        val imported = try {
            LegacyEncryptedWalletImporter.importTreasury(
                jsonBytes = jsonBytes,
                password = password,
                expectedAddress = DEVELOPMENT_TREASURY_ADDRESS,
            )
        } finally {
            password.fill('\u0000')
            jsonBytes.fill(0)
        }
        walletStore.saveTreasuryPrivateScalar(imported.privateScalar, imported.address)
    }

    fun isTreasuryWallet(): Boolean = walletStore.isTreasuryWallet()

    fun setPin(pin: String) = security.savePin(pin)
    fun verifyPin(pin: String): Boolean = security.verifyPin(pin)

    fun account(): LocalAccount {
        if (walletStore.isTreasuryWallet()) {
            val privateScalar = walletStore.loadTreasuryPrivateScalar()
            val key = SudharmaCrypto.keyFromPrivateScalar(privateScalar)
            val address = SudharmaCrypto.addressFromPublicKey(key.publicKey)
            require(address == DEVELOPMENT_TREASURY_ADDRESS) { "treasury wallet address mismatch" }
            require(walletStore.treasuryAddress() == address) { "stored treasury address mismatch" }
            return LocalAccount(address, privateScalar)
        }
        val phrase = walletStore.loadRecoveryPhrase()
        val metadata = walletStore.loadMetadata()
        val derived = SudharmaMobileDerivationV1.derive(
            RecoveryPhrase.seed(phrase),
            metadata.accountIndex,
        )
        require(derived.profile == metadata.profileId) { "wallet derivation profile mismatch" }
        return LocalAccount(derived.address, derived.privateScalar)
    }

    suspend fun balance(): AssetBalance = adapter().balance(account().address)

    suspend fun faucetInfo(): TestnetFaucetClient.Info = faucetClient().info()

    suspend fun requestInitialTestTokens(): TestnetFaucetClient.InitialGrant =
        faucetClient().requestInitial(account().address)

    suspend fun claimChallengeReward(transactionId: String): TestnetFaucetClient.ChallengeReward =
        faucetClient().claimChallenge(account().address, transactionId)

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

    suspend fun transactionStatus(transactionId: String): TransactionStatus = adapter().status(transactionId)

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
        return SudharmaChainAdapter(SudharmaRpcClient(preferences.rpcUrl))
    }

    private fun faucetClient(): TestnetFaucetClient {
        return TestnetFaucetClient(preferences.rpcUrl)
    }

    companion object {
        const val DEVELOPMENT_TREASURY_ADDRESS = "16d7dc9ec0495109007860a584c7cf9055da9abf"

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
