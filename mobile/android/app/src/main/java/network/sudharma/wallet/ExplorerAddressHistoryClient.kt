package network.sudharma.wallet

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass
import com.squareup.moshi.Moshi
import com.squareup.moshi.kotlin.reflect.KotlinJsonAdapterFactory
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.OkHttpClient
import okhttp3.Request
import java.io.IOException
import java.util.concurrent.TimeUnit

class ExplorerAddressHistoryClient(
    baseUrl: String,
    private val http: OkHttpClient = defaultHttpClient(),
) {
    private val root: HttpUrl = requireNotNull(baseUrl.toHttpUrlOrNull()) { "invalid explorer URL" }
    private val moshi = Moshi.Builder().add(KotlinJsonAdapterFactory()).build()

    suspend fun history(
        walletAddress: String,
        knownIds: Set<String> = emptySet(),
        nowMs: Long = System.currentTimeMillis(),
    ): List<WalletTransactionRecord> {
        require(walletAddress.matches(Regex("^[0-9a-f]{40}$"))) { "invalid address" }
        val url = root.newBuilder()
            .addPathSegments("v1/explorer/addresses")
            .addPathSegment(walletAddress)
            .addQueryParameter("limit", "100")
            .build()
        val dto = execute(Request.Builder().url(url).get().build())
        return dto.transactions
            .asSequence()
            .mapNotNull { item ->
                val tx = item.transaction
                if (tx.to != walletAddress || tx.from == walletAddress) return@mapNotNull null
                if (knownIds.contains(tx.id)) return@mapNotNull null
                if (!tx.id.matches(Regex("^[0-9a-f]{64}$"))) return@mapNotNull null
                if (!tx.from.matches(Regex("^[0-9a-f]{40}$"))) return@mapNotNull null
                if (tx.amount <= 0L || tx.fee < 0L) return@mapNotNull null
                val timestampMs = item.blockTimestamp
                    ?.takeIf { it > 0L }
                    ?.let { Math.multiplyExact(it, 1000L) }
                    ?: nowMs
                WalletTransactionRecord(
                    id = tx.id,
                    direction = TransactionDirection.RECEIVED,
                    amountAtomic = tx.amount,
                    counterparty = tx.from,
                    feeAtomic = tx.fee,
                    timestampMs = timestampMs,
                )
            }
            .sortedByDescending { it.timestampMs }
            .toList()
    }

    private suspend fun execute(request: Request): AddressDto = withContext(Dispatchers.IO) {
        try {
            http.newCall(request).execute().use { response ->
                val text = response.body?.string() ?: throw ExplorerHistoryException(response.code, "empty explorer response")
                if (!response.isSuccessful) {
                    val message = runCatching { moshi.adapter(ErrorDto::class.java).fromJson(text)?.error }.getOrNull()
                        ?: "HTTP ${response.code}"
                    throw ExplorerHistoryException(response.code, message)
                }
                try {
                    moshi.adapter(AddressDto::class.java).fromJson(text)
                        ?: throw ExplorerHistoryException(response.code, "invalid explorer response")
                } catch (error: ExplorerHistoryException) {
                    throw error
                } catch (error: Exception) {
                    throw ExplorerHistoryException(response.code, "invalid explorer response", error)
                }
            }
        } catch (error: ExplorerHistoryException) {
            throw error
        } catch (error: IOException) {
            throw ExplorerHistoryException(0, "explorer request failed", error)
        }
    }

    class ExplorerHistoryException(
        val statusCode: Int,
        override val message: String,
        cause: Throwable? = null,
    ) : Exception(message, cause)

    companion object {
        private fun defaultHttpClient(): OkHttpClient = OkHttpClient.Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(15, TimeUnit.SECONDS)
            .writeTimeout(10, TimeUnit.SECONDS)
            .build()
    }

    @JsonClass(generateAdapter = false)
    data class AddressDto(
        val address: String,
        val transactions: List<TransactionDto> = emptyList(),
    )

    @JsonClass(generateAdapter = false)
    data class TransactionDto(
        val transaction: TransactionViewDto,
        val status: String,
        @Json(name = "block_height") val blockHeight: Long? = null,
        @Json(name = "block_hash") val blockHash: String? = null,
        @Json(name = "block_timestamp") val blockTimestamp: Long? = null,
        val confirmations: Long = 0,
    )

    @JsonClass(generateAdapter = false)
    data class TransactionViewDto(
        val id: String,
        val from: String,
        val to: String,
        val amount: Long,
        val fee: Long,
        val nonce: Long,
    )

    @JsonClass(generateAdapter = false)
    data class ErrorDto(val error: String)
}
