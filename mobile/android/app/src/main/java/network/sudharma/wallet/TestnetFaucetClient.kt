package network.sudharma.wallet

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass
import com.squareup.moshi.Moshi
import com.squareup.moshi.kotlin.reflect.KotlinJsonAdapterFactory
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.IOException
import java.util.concurrent.TimeUnit

class TestnetFaucetClient(
    baseUrl: String,
    private val http: OkHttpClient = defaultHttpClient(),
) {
    private val root: HttpUrl = requireNotNull(baseUrl.toHttpUrlOrNull()) { "invalid faucet URL" }
    private val moshi = Moshi.Builder().add(KotlinJsonAdapterFactory()).build()

    data class Info(
        val enabled: Boolean,
        val challengeAddress: String,
        val initialGrantSudh: Int,
        val challengeSendSudh: Int,
        val challengeRewardSudh: Int,
        val maxRounds: Int,
        val cooldownHours: Int,
    )

    data class InitialGrant(
        val address: String,
        val amountSudh: Int,
        val transactionId: String,
        val status: String,
    )

    data class ChallengeReward(
        val address: String,
        val round: Int,
        val rewardSudh: Int,
        val rewardTransactionId: String,
        val nextEligibleAt: Long?,
        val status: String,
    )

    suspend fun info(): Info {
        val dto = execute(
            Request.Builder().url(root.newBuilder().addPathSegments("v1/faucet/info").build()).get().build(),
            InfoDto::class.java,
        )
        require(dto.challengeAddress.matches(Regex("^[0-9a-f]{40}$"))) { "invalid faucet challenge address" }
        return Info(
            dto.enabled,
            dto.challengeAddress,
            dto.initialGrantSudh,
            dto.challengeSendSudh,
            dto.challengeRewardSudh,
            dto.maxRounds,
            dto.cooldownHours,
        )
    }

    suspend fun requestInitial(address: String): InitialGrant {
        require(address.matches(Regex("^[0-9a-f]{40}$"))) { "invalid Sudharma address" }
        val dto = post(
            "v1/faucet/request",
            InitialRequest(address),
            InitialGrantDto::class.java,
        )
        return InitialGrant(dto.address, dto.amountSudh, dto.transactionId, dto.status)
    }

    suspend fun claimChallenge(address: String, transactionId: String): ChallengeReward {
        require(address.matches(Regex("^[0-9a-f]{40}$"))) { "invalid Sudharma address" }
        require(transactionId.matches(Regex("^[0-9a-f]{64}$"))) { "invalid transaction ID" }
        val dto = post(
            "v1/faucet/challenge",
            ChallengeRequest(address, transactionId),
            ChallengeRewardDto::class.java,
        )
        return ChallengeReward(
            dto.address,
            dto.round,
            dto.rewardSudh,
            dto.rewardTransactionId,
            dto.nextEligibleAt,
            dto.status,
        )
    }

    private suspend fun <I : Any, O> post(path: String, input: I, outputType: Class<O>): O {
        @Suppress("UNCHECKED_CAST")
        val adapter = moshi.adapter(input.javaClass as Class<I>)
        val json = adapter.toJson(input)
        val request = Request.Builder()
            .url(root.newBuilder().addPathSegments(path).build())
            .post(json.toRequestBody("application/json; charset=utf-8".toMediaType()))
            .build()
        return execute(request, outputType)
    }

    private suspend fun <T> execute(request: Request, type: Class<T>): T = withContext(Dispatchers.IO) {
        try {
            http.newCall(request).execute().use { response ->
                val text = response.body?.string() ?: throw FaucetException(response.code, "empty faucet response")
                if (!response.isSuccessful) {
                    val message = runCatching { moshi.adapter(ErrorDto::class.java).fromJson(text)?.error }.getOrNull()
                        ?: "HTTP ${response.code}"
                    throw FaucetException(response.code, message)
                }
                try {
                    moshi.adapter(type).fromJson(text)
                        ?: throw FaucetException(response.code, "invalid faucet response")
                } catch (error: FaucetException) {
                    throw error
                } catch (error: Exception) {
                    throw FaucetException(response.code, "invalid faucet response", error)
                }
            }
        } catch (error: FaucetException) {
            throw error
        } catch (error: IOException) {
            throw FaucetException(0, "faucet request failed", error)
        }
    }

    class FaucetException(
        val statusCode: Int,
        override val message: String,
        cause: Throwable? = null,
    ) : Exception(message, cause)

    companion object {
        private fun defaultHttpClient(): OkHttpClient = OkHttpClient.Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(15, TimeUnit.SECONDS)
            .writeTimeout(15, TimeUnit.SECONDS)
            .build()
    }

    @JsonClass(generateAdapter = false)
    data class InfoDto(
        val enabled: Boolean,
        @Json(name = "challenge_address") val challengeAddress: String,
        @Json(name = "initial_grant_sudh") val initialGrantSudh: Int,
        @Json(name = "challenge_send_sudh") val challengeSendSudh: Int,
        @Json(name = "challenge_reward_sudh") val challengeRewardSudh: Int,
        @Json(name = "max_rounds") val maxRounds: Int,
        @Json(name = "cooldown_hours") val cooldownHours: Int,
        @Json(name = "testnet_only") val testnetOnly: Boolean,
    )

    @JsonClass(generateAdapter = false)
    data class InitialRequest(val address: String)

    @JsonClass(generateAdapter = false)
    data class ChallengeRequest(
        val address: String,
        @Json(name = "transaction_id") val transactionId: String,
    )

    @JsonClass(generateAdapter = false)
    data class InitialGrantDto(
        val address: String,
        @Json(name = "amount_sudh") val amountSudh: Int,
        @Json(name = "transaction_id") val transactionId: String,
        val status: String,
    )

    @JsonClass(generateAdapter = false)
    data class ChallengeRewardDto(
        val address: String,
        val round: Int,
        @Json(name = "reward_sudh") val rewardSudh: Int,
        @Json(name = "reward_transaction_id") val rewardTransactionId: String,
        @Json(name = "next_eligible_at") val nextEligibleAt: Long?,
        val status: String,
    )

    @JsonClass(generateAdapter = false)
    data class ErrorDto(val error: String)
}
