package network.sudharma.wallet

import android.content.Context
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

class WalletPreferences(context: Context) {
    private val prefs = context.getSharedPreferences("sudharma_wallet_app_v1", Context.MODE_PRIVATE)

    var rpcUrl: String
        get() = prefs.getString("testnet_rpc_url", DEFAULT_TESTNET_RPC_URL) ?: DEFAULT_TESTNET_RPC_URL
        set(value) {
            val trimmed = value.trim().trimEnd('/')
            if (trimmed.isNotEmpty()) validateRpcUrl(trimmed)
            prefs.edit().putString("testnet_rpc_url", trimmed).apply()
        }

    var backupAcknowledged: Boolean
        get() = prefs.getBoolean("backup_acknowledged", false)
        set(value) { prefs.edit().putBoolean("backup_acknowledged", value).apply() }

    var lastSyncedChainHeight: Long
        get() = prefs.getLong("last_synced_chain_height", 0L)
        set(value) { prefs.edit().putLong("last_synced_chain_height", value).apply() }

    fun addTransactionRecord(record: WalletTransactionRecord) {
        val current = transactionRecords().toMutableList()
        current.removeAll { it.id == record.id }
        current.add(0, record)
        saveRecords(current.take(MAX_RECORDS))
        addTransactionId(record.id)
    }

    fun addTransactionId(id: String) {
        if (!id.matches(Regex("^[0-9a-f]{64}$"))) return
        val current = transactionIds().toMutableList()
        current.remove(id)
        current.add(0, id)
        prefs.edit().putString("tx_ids", current.take(30).joinToString(",")).apply()
    }

    fun transactionRecords(): List<WalletTransactionRecord> {
        migrateLegacyIdsIfNeeded()
        val encoded = prefs.getString("tx_records", "") ?: ""
        if (encoded.isBlank()) return emptyList()
        return encoded.lineSequence()
            .mapNotNull(WalletTransactionRecord::decode)
            .toList()
    }

    fun transactionIds(): List<String> = transactionRecords().map { it.id }.ifEmpty {
        (prefs.getString("tx_ids", "") ?: "")
            .split(',')
            .filter { it.matches(Regex("^[0-9a-f]{64}$")) }
    }

    fun clear() {
        prefs.edit().clear().apply()
    }

    private fun saveRecords(records: List<WalletTransactionRecord>) {
        prefs.edit().putString("tx_records", records.joinToString("\n") { it.encode() }).apply()
    }

    private fun migrateLegacyIdsIfNeeded() {
        if (prefs.contains("tx_records")) return
        val legacy = (prefs.getString("tx_ids", "") ?: "")
            .split(',')
            .filter { it.matches(Regex("^[0-9a-f]{64}$")) }
        if (legacy.isEmpty()) return
        val migrated = legacy.map { id ->
            WalletTransactionRecord(
                id = id,
                direction = TransactionDirection.SENT,
                amountAtomic = 1L,
                counterparty = PLACEHOLDER_COUNTERPARTY,
                timestampMs = 0L,
            )
        }
        saveRecords(migrated)
    }

    companion object {
        const val DEFAULT_TESTNET_RPC_URL =
            "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com"
        private const val MAX_RECORDS = 50

        fun validateRpcUrl(value: String) {
            val url = value.toHttpUrlOrNull() ?: throw IllegalArgumentException("Invalid RPC URL")
            require(url.scheme == "https" || (BuildConfig.DEBUG && url.scheme == "http")) {
                "RPC must use HTTPS"
            }
        }
    }
}
