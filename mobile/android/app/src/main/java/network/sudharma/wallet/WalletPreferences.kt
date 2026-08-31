package network.sudharma.wallet

import android.content.Context
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

class WalletPreferences(context: Context) {
    private val prefs = context.getSharedPreferences("sudharma_wallet_app_v1", Context.MODE_PRIVATE)

    var rpcUrl: String
        get() {
            val stored = prefs.getString(RPC_URL_KEY, null)
            val resolved = RpcEndpointPolicy.resolve(stored, allowDebugOverride = BuildConfig.DEBUG)
            if (stored != resolved) prefs.edit().putString(RPC_URL_KEY, resolved).apply()
            return resolved
        }
        set(value) {
            if (!BuildConfig.DEBUG) {
                prefs.edit().putString(RPC_URL_KEY, TestnetChallengeConfig.DEFAULT_RPC_URL).apply()
                return
            }
            val trimmed = value.trim().trimEnd('/')
            if (trimmed.isNotEmpty()) validateRpcUrl(trimmed)
            if (trimmed.isEmpty()) {
                prefs.edit().remove(RPC_URL_KEY).apply()
            } else {
                prefs.edit().putString(RPC_URL_KEY, trimmed).apply()
            }
        }

    var backupAcknowledged: Boolean
        get() = prefs.getBoolean("backup_acknowledged", false)
        set(value) { prefs.edit().putBoolean("backup_acknowledged", value).apply() }

    fun addTransactionId(id: String) {
        if (!id.matches(Regex("^[0-9a-f]{64}$"))) return
        val current = transactionIds().toMutableList()
        current.remove(id)
        current.add(0, id)
        prefs.edit().putString("tx_ids", current.take(30).joinToString(",")).apply()
    }

    fun transactionIds(): List<String> = (prefs.getString("tx_ids", "") ?: "")
        .split(',')
        .filter { it.matches(Regex("^[0-9a-f]{64}$")) }

    fun clear() = prefs.edit().clear().apply()

    companion object {
        private const val RPC_URL_KEY = "testnet_rpc_url"

        fun validateRpcUrl(value: String) {
            val url = value.toHttpUrlOrNull() ?: throw IllegalArgumentException("Invalid RPC URL")
            require(url.scheme == "https" || (BuildConfig.DEBUG && url.scheme == "http")) {
                "RPC must use HTTPS"
            }
        }
    }
}
