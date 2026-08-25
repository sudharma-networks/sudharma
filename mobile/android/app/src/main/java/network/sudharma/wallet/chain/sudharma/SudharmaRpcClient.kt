package network.sudharma.wallet.chain.sudharma

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass
import com.squareup.moshi.Moshi
import com.squareup.moshi.kotlin.reflect.KotlinJsonAdapterFactory
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.HttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import java.io.IOException
import java.util.Base64
import java.util.concurrent.TimeUnit

class SudharmaRpcClient(
    baseUrl: String,
    private val http: OkHttpClient = defaultHttpClient(),
) {
    private val root: HttpUrl = requireNotNull(baseUrl.toHttpUrlOrNull()) { "invalid RPC URL" }
    private val moshi = Moshi.Builder().add(KotlinJsonAdapterFactory()).build()

    data class Status(
        val network: String,
        val symbol: String,
        val height: Long,
        val tipHash: String,
        val peers: Int,
        val mempool: Int,
        val issuedSupply: Long,
    )

    data class Account(
        val address: String,
        val balance: Long,
        val confirmedNonce: Long,
        val nextNonce: Long,
    )

    data class SubmitResult(val transactionId: String, val relayedPeers: Int, val accepted: Boolean)

    data class RemoteTransactionStatus(
        val status: String,
        val blockHeight: Long?,
        val blockHash: String?,
        val confirmations: Long,
    )

    suspend fun status(): Status {
        val dto = get("v1/status", StatusDto::class.java)
        return Status(dto.network, dto.symbol, dto.height, dto.tipHash, dto.peers, dto.mempool, dto.issuedSupply)
    }

    suspend fun account(address: String): Account {
        require(address.isNotBlank() && address.length <= 256) { "invalid address" }
        val url = root.newBuilder().addPathSegments("v1/accounts").addPathSegment(address).build()
        val dto = execute(Request.Builder().url(url).get().build(), AccountDto::class.java)
        return Account(dto.address, dto.balance, dto.confirmedNonce, dto.nextNonce)
    }

    suspend fun submit(tx: SudharmaTransaction): SubmitResult {
        require(tx.verify()) { "transaction must be signed and valid" }
        val wire = TransactionWire(
            id = tx.id,
            from = tx.from,
            to = tx.to,
            amount = tx.amount,
            fee = tx.fee,
            nonce = tx.nonce,
            publicKey = Base64.getEncoder().encodeToString(requireNotNull(tx.publicKey)),
            signature = Base64.getEncoder().encodeToString(requireNotNull(tx.signature)),
        )
        val json = requireNotNull(moshi.adapter(TransactionWire::class.java).toJson(wire))
        val request = Request.Builder()
            .url(root.newBuilder().addPathSegments("v1/transactions").build())
            .post(json.toRequestBody("application/json; charset=utf-8".toMediaType()))
            .build()
        val dto = execute(request, SubmitDto::class.java)
        return SubmitResult(dto.transactionId, dto.relayedPeers, dto.accepted)
    }

    suspend fun transaction(transactionId: String): RemoteTransactionStatus {
        require(transactionId.matches(Regex("^[0-9a-f]{64}$"))) { "invalid transaction ID" }
        val url = root.newBuilder().addPathSegments("v1/transactions").addPathSegment(transactionId).build()
        val dto = execute(Request.Builder().url(url).get().build(), TransactionStatusDto::class.java)
        return RemoteTransactionStatus(dto.status, dto.blockHeight, dto.blockHash, dto.confirmations)
    }

    private suspend fun <T> get(path: String, type: Class<T>): T {
        val request = Request.Builder().url(root.newBuilder().addPathSegments(path).build()).get().build()
        return execute(request, type)
    }

    private suspend fun <T> execute(request: Request, type: Class<T>): T = withContext(Dispatchers.IO) {
        try {
            http.newCall(request).execute().use { response ->
                val body = response.body ?: throw RpcException(response.code, "empty RPC response")
                if (body.contentLength() > MAX_RESPONSE_BYTES) {
                    throw RpcException(response.code, "RPC response too large")
                }
                val bytes = body.source().readByteArray(MAX_RESPONSE_BYTES + 1)
                if (bytes.size.toLong() > MAX_RESPONSE_BYTES) {
                    throw RpcException(response.code, "RPC response too large")
                }
                val text = bytes.toString(Charsets.UTF_8)
                if (!response.isSuccessful) {
                    val message = runCatching {
                        moshi.adapter(ErrorDto::class.java).fromJson(text)?.error
                    }.getOrNull() ?: "HTTP ${response.code}"
                    throw RpcException(response.code, message)
                }
                try {
                    moshi.adapter(type).fromJson(text)
                        ?: throw RpcException(response.code, "invalid RPC response")
                } catch (error: RpcException) {
                    throw error
                } catch (error: Exception) {
                    throw RpcException(response.code, "invalid RPC response", error)
                }
            }
        } catch (error: RpcException) {
            throw error
        } catch (error: IOException) {
            throw RpcException(0, "RPC request failed", error)
        }
    }

    class RpcException(
        val statusCode: Int,
        override val message: String,
        cause: Throwable? = null,
    ) : Exception(message, cause)

    companion object {
        private const val MAX_RESPONSE_BYTES = 4 * 1024 * 1024L

        private fun defaultHttpClient(): OkHttpClient = OkHttpClient.Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(10, TimeUnit.SECONDS)
            .writeTimeout(10, TimeUnit.SECONDS)
            .build()
    }

    @JsonClass(generateAdapter = false)
    data class StatusDto(
        val network: String,
        val coin: String,
        val symbol: String,
        @Json(name = "node_id") val nodeId: String,
        @Json(name = "p2p_address") val p2pAddress: String,
        val height: Long,
        @Json(name = "tip_hash") val tipHash: String,
        @Json(name = "total_work") val totalWork: String,
        val peers: Int,
        val mempool: Int,
        @Json(name = "issued_supply") val issuedSupply: Long,
    )

    @JsonClass(generateAdapter = false)
    data class AccountDto(
        val address: String,
        val balance: Long,
        @Json(name = "confirmed_nonce") val confirmedNonce: Long,
        @Json(name = "next_nonce") val nextNonce: Long,
    )

    @JsonClass(generateAdapter = false)
    data class SubmitDto(
        @Json(name = "transaction_id") val transactionId: String,
        @Json(name = "relayed_peers") val relayedPeers: Int,
        val accepted: Boolean,
    )

    @JsonClass(generateAdapter = false)
    data class TransactionStatusDto(
        val status: String,
        @Json(name = "block_height") val blockHeight: Long? = null,
        @Json(name = "block_hash") val blockHash: String? = null,
        val confirmations: Long = 0,
    )

    @JsonClass(generateAdapter = false)
    data class ErrorDto(val error: String)

    @JsonClass(generateAdapter = false)
    data class TransactionWire(
        @Json(name = "ID") val id: String,
        @Json(name = "From") val from: String,
        @Json(name = "To") val to: String,
        @Json(name = "Amount") val amount: Long,
        @Json(name = "Fee") val fee: Long,
        @Json(name = "Nonce") val nonce: Long,
        @Json(name = "PublicKey") val publicKey: String,
        @Json(name = "Signature") val signature: String,
    )
}
