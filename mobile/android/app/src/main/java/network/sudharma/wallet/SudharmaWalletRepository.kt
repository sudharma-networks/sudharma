package network.sudharma.wallet

import android.content.Context
import network.sudharma.wallet.chain.AssetBalance
import network.sudharma.wallet.chain.TransactionState
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
    @Volatile private var lastFaucetInfo: TestnetFaucetClient.Info? = null

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
        val metadata = walletStore.loadMetadata()
        val derived = SudharmaMobileDerivationV1.derive(
            RecoveryPhrase.seed(phrase),
            metadata.accountIndex,
        )
        require(derived.profile == metadata.profileId) { "wallet derivation profile mismatch" }
        return LocalAccount(derived.address, derived.privateScalar)
    }

    suspend fun balance(): AssetBalance = adapter().balance(account().address)

    suspend fun faucetInfo(): TestnetFaucetClient.Info = faucetClient().info().also { lastFaucetInfo = it }

    suspend fun requestInitialTestTokens(): TestnetFaucetClient.InitialGrant =
        faucetClient().requestInitial(account().address)

    suspend fun claimChallengeReward(transactionId: String): TestnetFaucetClient.ChallengeReward =
        faucetClient().claimChallenge(account().address, transactionId)

    suspend fun send(to: String, amountText: String, challengeMode: Boolean = false): TransactionStatus {
        val account = account()
        val adapter = adapter()
        require(adapter.validateAddress(to)) { "Invalid Sudharma address" }
        require(to != account.address) { "Cannot send to the same wallet" }
        val amount = parseCoinAmount(amountText)
        if (challengeMode) {
            val info = faucetInfo()
            require(TestnetChallengePolicy.matchesOfficialChallenge(info, to, amount)) {
                "Challenge details changed; reopen the challenge and try again"
            }
        }
        val remoteAccount = adapter.balance(account.address)
        val fee = adapter.estimateFee(amount).feeAtomic
        require(remoteAccount.amount.atomic >= Math.addExact(amount, fee)) { "Insufficient balance including fee" }
        val unsigned = adapter.unsigned(account.address, to, amount, remoteAccount.nextNonce)
        val signed = adapter.sign(unsigned, account.privateScalar)
        val status = adapter.submit(signed)
        preferences.addTransactionRecord(
            WalletTransactionRecord(
                id = status.id,
                direction = TransactionDirection.SENT,
                amountAtomic = amount,
                counterparty = to,
                feeAtomic = fee,
                timestampMs = System.currentTimeMillis(),
            ),
        )
        return status
    }

    suspend fun activityHistory(): List<WalletActivityItem> {
        syncReceivedTransactions()
        val account = account()
        val rpc = rpcClient()
        val rawRecords = preferences.transactionRecords()
        val enrichedRecords = TransactionRecordEnricher.enrich(
            records = rawRecords,
            walletAddress = account.address,
            fetchStatus = rpc::transaction,
        )
        enrichedRecords.forEach { enriched ->
            val previous = rawRecords.firstOrNull { it.id == enriched.id }
            if (previous != null && previous != enriched) {
                preferences.addTransactionRecord(enriched)
            }
        }
        val adapter = adapter()
        return TransactionActivityLoader.load(enrichedRecords, adapter::status)
    }

    suspend fun syncReceivedTransactions() {
        val account = account()
        val rpc = rpcClient()
        val network = rpc.status()
        val known = preferences.transactionRecords().map { it.id }.toSet()
        val incoming = ReceivedTransactionScanner.scan(
            rpc = rpc,
            address = account.address,
            knownIds = known,
            chainHeight = network.height,
        )
        incoming.forEach { preferences.addTransactionRecord(it) }
        preferences.lastSyncedChainHeight = network.height
    }

    suspend fun transactionStatus(transactionId: String): TransactionStatus = adapter().status(transactionId)

    suspend fun transactionConfirmed(transactionId: String): Boolean =
        transactionStatus(transactionId).state == TransactionState.CONFIRMED

    suspend fun transactionStatuses(): List<TransactionStatus> =
        activityHistory().map {
            TransactionStatus(it.record.id, it.state, it.confirmations)
        }

    fun resetWallet() {
        walletStore.clear()
        security.clear()
        preferences.clear()
        lastFaucetInfo = null
    }

    private fun adapter(): SudharmaChainAdapter {
        val url = preferences.rpcUrl
        require(url.isNotBlank()) { "Sudharma Testnet RPC is not configured" }
        return SudharmaChainAdapter(rpcClient())
    }

    private fun rpcClient(): SudharmaRpcClient {
        val url = preferences.rpcUrl
        require(url.isNotBlank()) { "Sudharma Testnet RPC is not configured" }
        return SudharmaRpcClient(url)
    }

    private fun faucetClient(): TestnetFaucetClient {
        val url = preferences.rpcUrl
        require(url.isNotBlank()) { "Sudharma Testnet RPC is not configured" }
        return TestnetFaucetClient(url)
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
