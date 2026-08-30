from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
WALLET = ROOT / "mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt"
REPO = ROOT / "mobile/android/app/src/main/java/network/sudharma/wallet/SudharmaWalletRepository.kt"
GRADLE = ROOT / "mobile/android/app/build.gradle.kts"
PKG = ROOT / "mobile/android/app/src/main/java/network/sudharma/wallet"


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly one anchor, found {count}")
    return text.replace(old, new, 1)


# Wallet UI integration.
w = WALLET.read_text()
w = replace_once(
    w,
    "import androidx.compose.foundation.background\n",
    "import androidx.compose.foundation.background\nimport androidx.compose.foundation.clickable\n",
    "clickable import",
)
w = replace_once(
    w,
    "import androidx.compose.foundation.text.KeyboardOptions\n",
    "import androidx.compose.foundation.text.KeyboardOptions\nimport androidx.compose.foundation.text.selection.SelectionContainer\n",
    "selection import",
)
w = replace_once(
    w,
    "import androidx.compose.ui.text.style.TextAlign\n",
    "import androidx.compose.ui.text.style.TextAlign\nimport androidx.compose.ui.text.style.TextDecoration\n",
    "text decoration import",
)
w = replace_once(
    w,
    "    var recipient by remember { mutableStateOf(\"\") }\n    var amount by remember { mutableStateOf(\"\") }\n    var confirm by remember { mutableStateOf(false) }\n",
    "    var recipient by remember { mutableStateOf(\"\") }\n    var amount by remember { mutableStateOf(\"\") }\n    var challengeMode by remember { mutableStateOf(false) }\n    var challengeInfo by remember { mutableStateOf<TestnetFaucetClient.Info?>(null) }\n    var challengeMessage by remember { mutableStateOf(\"\") }\n    var claiming by remember { mutableStateOf(false) }\n    var confirm by remember { mutableStateOf(false) }\n",
    "challenge state",
)
w = replace_once(
    w,
    "    var result by remember { mutableStateOf<TransactionStatus?>(null) }\n    var sending by remember { mutableStateOf(false) }\n\n    val scanner = rememberLauncherForActivityResult(ScanContract()) { scan ->\n",
    "    var result by remember { mutableStateOf<TransactionStatus?>(null) }\n    var sending by remember { mutableStateOf(false) }\n\n    LaunchedEffect(repository.preferences.rpcUrl) {\n        runCatching { repository.faucetInfo() }\n            .onSuccess { challengeInfo = it }\n            .onFailure { challengeInfo = null }\n    }\n\n    val scanner = rememberLauncherForActivityResult(ScanContract()) { scan ->\n",
    "challenge info load",
)
w = replace_once(
    w,
    "            runCatching { repository.send(recipient, amount) }\n",
    "            runCatching { repository.send(recipient, amount, challengeMode = challengeMode) }\n",
    "challenge send call",
)
w = replace_once(
    w,
    "            TransactionReferenceActions(it.id)\n            Text(\"Saved to Activity. Refresh Home to update confirmations.\", style = MaterialTheme.typography.bodySmall)\n",
    "            TransactionReferenceActions(it.id)\n            if (challengeMode) {\n                Text(\"Challenge payment submitted. The 50 Test SUDH reward can be claimed after this transaction is confirmed.\")\n                Button(\n                    enabled = !claiming,\n                    onClick = {\n                        claiming = true\n                        challengeMessage = \"Checking transaction confirmation…\"\n                        scope.launch {\n                            runCatching {\n                                val latest = repository.transactionStatus(it.id)\n                                require(latest.state == TransactionState.CONFIRMED) {\n                                    \"Transaction is not confirmed yet. Try again after the next testnet block.\"\n                                }\n                                repository.claimChallengeReward(it.id)\n                            }.onSuccess { reward ->\n                                challengeMessage = \"Round ${reward.round} complete — ${reward.rewardSudh} Test SUDH reward submitted.\"\n                            }.onFailure { failure ->\n                                challengeMessage = failure.message ?: \"Unable to claim challenge reward\"\n                            }\n                            claiming = false\n                        }\n                    },\n                    modifier = Modifier.fillMaxWidth(),\n                ) {\n                    Text(if (claiming) \"Checking…\" else \"Check Confirmation & Claim ${challengeInfo?.challengeRewardSudh ?: 50} Test SUDH\")\n                }\n                if (challengeMessage.isNotEmpty()) Text(challengeMessage, style = MaterialTheme.typography.bodySmall)\n            }\n            Text(\"Saved to Activity. Refresh Home to update confirmations.\", style = MaterialTheme.typography.bodySmall)\n",
    "challenge claim action",
)
w = replace_once(
    w,
    "        if (!confirm) {\n            OutlinedTextField(recipient, { recipient = it.trim() }, label = { Text(\"Recipient address\") }, modifier = Modifier.fillMaxWidth())\n",
    "        if (!confirm) {\n            Card(Modifier.fillMaxWidth()) {\n                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {\n                    Text(\"Testnet Challenge\", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)\n                    Text(\"Send ${challengeInfo?.challengeSendSudh ?: 25} test SUDH to the official challenge address. After confirmation, eligible testers receive ${challengeInfo?.challengeRewardSudh ?: 50} test SUDH back. Up to ${challengeInfo?.maxRounds ?: 5} rounds with a ${challengeInfo?.cooldownHours ?: 24}-hour wait between successful rounds.\")\n                    Button(\n                        enabled = challengeInfo?.enabled == true && challengeInfo?.challengeAddress?.matches(Regex(\"^[0-9a-f]{40}$\")) == true,\n                        onClick = {\n                            challengeMode = true\n                            recipient = challengeInfo?.challengeAddress.orEmpty()\n                            amount = (challengeInfo?.challengeSendSudh ?: 25).toString()\n                            error = \"\"\n                        },\n                        modifier = Modifier.fillMaxWidth(),\n                    ) { Text(\"Use 25 → 50 Testnet Challenge\") }\n                    if (challengeInfo?.enabled != true) {\n                        Text(\"Official challenge service is currently unavailable. It will enable automatically when the protected faucet is online.\", style = MaterialTheme.typography.bodySmall)\n                    }\n                    Text(\"TESTNET ONLY — NO MONETARY VALUE\", style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.Bold)\n                }\n            }\n            if (challengeMode) {\n                Text(\"Challenge mode — official address and 25 SUDH amount are prefilled.\", style = MaterialTheme.typography.bodySmall, fontWeight = FontWeight.Bold)\n            }\n            OutlinedTextField(recipient, { recipient = it.trim(); challengeMode = false }, label = { Text(\"Recipient address\") }, modifier = Modifier.fillMaxWidth())\n",
    "challenge card",
)
w = replace_once(
    w,
    "                { amount = it },\n                label = { Text(\"Amount (SUDH)\") },\n",
    "                { amount = it; challengeMode = false },\n                label = { Text(\"Amount (SUDH)\") },\n",
    "challenge amount edit reset",
)
w = replace_once(
    w,
    "                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {\n                    Text(\"Recipient: ${recipient.take(10)}…${recipient.takeLast(8)}\")\n",
    "                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {\n                    if (challengeMode) Text(\"Testnet Challenge — Send 25 / Receive 50\", fontWeight = FontWeight.Bold)\n                    Text(\"Recipient: ${recipient.take(10)}…${recipient.takeLast(8)}\")\n",
    "challenge review label",
)
old_refs = '''@Composable
private fun TransactionReferenceActions(transactionId: String) {
    val context = LocalContext.current
    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(transactionId, style = MaterialTheme.typography.bodySmall)
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedButton(onClick = {
                val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                clipboard.setPrimaryClip(ClipData.newPlainText("Sudharma transaction ID", transactionId))
            }, modifier = Modifier.weight(1f)) { Text("Copy ID") }
            OutlinedButton(onClick = {
                context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(ExplorerLinks.transactionUrl(transactionId))))
            }, modifier = Modifier.weight(1f)) { Text("Explorer") }
        }
    }
}
'''
new_refs = '''@Composable
private fun TransactionReferenceActions(transactionId: String) {
    val context = LocalContext.current
    val explorerUrl = ExplorerLinks.transactionUrl(transactionId)
    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text("TX ID — tap to verify in Explorer", style = MaterialTheme.typography.labelMedium)
        SelectionContainer {
            Text(
                transactionId,
                modifier = Modifier.clickable {
                    context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(explorerUrl)))
                },
                style = MaterialTheme.typography.bodySmall.copy(
                    color = MaterialTheme.colorScheme.primary,
                    textDecoration = TextDecoration.Underline,
                ),
            )
        }
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedButton(onClick = {
                val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                clipboard.setPrimaryClip(ClipData.newPlainText("Sudharma transaction ID", transactionId))
            }, modifier = Modifier.weight(1f)) { Text("Copy TX ID") }
            OutlinedButton(onClick = {
                context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(ExplorerLinks.transactionUrl(transactionId))))
            }, modifier = Modifier.weight(1f)) { Text("Open Explorer ↗") }
        }
    }
}
'''
w = replace_once(w, old_refs, new_refs, "transaction reference actions")
WALLET.write_text(w)

# Repository integration: preserve detailed local history while adding challenge validation/claiming.
r = REPO.read_text()
r = replace_once(
    r,
    "import network.sudharma.wallet.chain.AssetBalance\nimport network.sudharma.wallet.chain.TransactionStatus\n",
    "import network.sudharma.wallet.chain.AssetBalance\nimport network.sudharma.wallet.chain.TransactionState\nimport network.sudharma.wallet.chain.TransactionStatus\n",
    "repository transaction state import",
)
r = replace_once(
    r,
    "    val preferences = WalletPreferences(context)\n\n    data class LocalAccount",
    "    val preferences = WalletPreferences(context)\n    @Volatile private var lastFaucetInfo: TestnetFaucetClient.Info? = null\n\n    data class LocalAccount",
    "faucet cache",
)
r = replace_once(
    r,
    "    suspend fun balance(): AssetBalance = adapter().balance(account().address)\n\n    suspend fun send(to: String, amountText: String): TransactionStatus {\n",
    "    suspend fun balance(): AssetBalance = adapter().balance(account().address)\n\n    suspend fun faucetInfo(): TestnetFaucetClient.Info = faucetClient().info().also { lastFaucetInfo = it }\n\n    suspend fun claimChallengeReward(transactionId: String): TestnetFaucetClient.ChallengeReward =\n        faucetClient().claimChallenge(account().address, transactionId)\n\n    suspend fun send(to: String, amountText: String, challengeMode: Boolean = false): TransactionStatus {\n",
    "repository challenge methods",
)
r = replace_once(
    r,
    "        val amount = parseCoinAmount(amountText)\n        val remoteAccount = adapter.balance(account.address)\n",
    "        val amount = parseCoinAmount(amountText)\n        if (challengeMode) {\n            val info = faucetInfo()\n            require(TestnetChallengePolicy.matchesOfficialChallenge(info, to, amount)) {\n                \"Challenge details changed; reopen the challenge and try again\"\n            }\n        }\n        val remoteAccount = adapter.balance(account.address)\n",
    "challenge validation",
)
r = replace_once(
    r,
    "    suspend fun transactionStatuses(): List<TransactionStatus> =\n        activityHistory().map {\n",
    "    suspend fun transactionStatus(transactionId: String): TransactionStatus = adapter().status(transactionId)\n\n    suspend fun transactionConfirmed(transactionId: String): Boolean =\n        transactionStatus(transactionId).state == TransactionState.CONFIRMED\n\n    suspend fun transactionStatuses(): List<TransactionStatus> =\n        activityHistory().map {\n",
    "transaction status method",
)
r = replace_once(
    r,
    "        preferences.clear()\n    }\n\n    private fun adapter(): SudharmaChainAdapter {\n",
    "        preferences.clear()\n        lastFaucetInfo = null\n    }\n\n    private fun adapter(): SudharmaChainAdapter {\n",
    "clear faucet cache",
)
r = replace_once(
    r,
    "    private fun rpcClient(): SudharmaRpcClient {\n        val url = preferences.rpcUrl\n        require(url.isNotBlank()) { \"Sudharma Testnet RPC is not configured\" }\n        return SudharmaRpcClient(url)\n    }\n\n    companion object {\n",
    "    private fun rpcClient(): SudharmaRpcClient {\n        val url = preferences.rpcUrl\n        require(url.isNotBlank()) { \"Sudharma Testnet RPC is not configured\" }\n        return SudharmaRpcClient(url)\n    }\n\n    private fun faucetClient(): TestnetFaucetClient {\n        val url = preferences.rpcUrl\n        require(url.isNotBlank()) { \"Sudharma Testnet RPC is not configured\" }\n        return TestnetFaucetClient(url)\n    }\n\n    companion object {\n",
    "faucet client",
)
REPO.write_text(r)

# Challenge API client and validation policy copied from the already-proven testnet challenge implementation.
(PKG / "TestnetFaucetClient.kt").write_text(r'''package network.sudharma.wallet

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
    )

    @JsonClass(generateAdapter = false)
    data class ChallengeRequest(
        val address: String,
        @Json(name = "transaction_id") val transactionId: String,
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
''')

(PKG / "TestnetChallengePolicy.kt").write_text(r'''package network.sudharma.wallet

object TestnetChallengePolicy {
    fun matchesOfficialChallenge(
        info: TestnetFaucetClient.Info?,
        to: String,
        amountAtomic: Long,
        coinAtomic: Long = 100_000_000L,
    ): Boolean {
        if (info?.enabled != true) return false
        val challengeAmount = runCatching {
            Math.multiplyExact(info.challengeSendSudh.toLong(), coinAtomic)
        }.getOrNull() ?: return false
        return to == info.challengeAddress && amountAtomic == challengeAmount
    }
}
''')

# Version bump only after requirements exist in source.
g = GRADLE.read_text()
g = replace_once(g, '        versionCode = 4\n        versionName = "0.1.3-testnet"\n', '        versionCode = 5\n        versionName = "0.1.4-testnet"\n', "version bump")
GRADLE.write_text(g)

print("wallet 0.1.4 integration patch applied")
