from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PKG = ROOT / "mobile/android/app/src/main/java/network/sudharma/wallet"
WALLET = PKG / "WalletApp.kt"
REPO = PKG / "SudharmaWalletRepository.kt"
FAUCET = PKG / "TestnetFaucetClient.kt"
LOADER = PKG / "TransactionActivityLoader.kt"
PRESENTATION = PKG / "TransactionDetailPresentation.kt"


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly one anchor, found {count}")
    return text.replace(old, new, 1)


# Activity history must always be newest first.
LOADER.write_text('''package network.sudharma.wallet

import network.sudharma.wallet.chain.TransactionStatus

internal object TransactionActivityLoader {
    suspend fun load(
        records: List<WalletTransactionRecord>,
        fetch: suspend (String) -> TransactionStatus,
    ): List<WalletActivityItem> = records
        .sortedByDescending { it.timestampMs }
        .map { record ->
            val status = fetch(record.id)
            WalletActivityItem(record = record, state = status.state, confirmations = status.confirmations)
        }
}
''')

# Pure presentation model keeps the full detail screen deterministic and testable.
PRESENTATION.write_text('''package network.sudharma.wallet

data class TransactionDetailPresentation(
    val direction: String,
    val amount: String,
    val counterpartyLabel: String,
    val counterparty: String,
    val networkFee: String?,
    val dateTime: String,
    val status: String,
    val confirmations: Long,
    val transactionId: String,
    val explorerUrl: String,
) {
    companion object {
        fun from(item: WalletActivityItem): TransactionDetailPresentation {
            val record = item.record
            return TransactionDetailPresentation(
                direction = TransactionDetailFormatter.directionLabel(record.direction),
                amount = if (TransactionDetailFormatter.hasKnownAmount(record)) {
                    TransactionDetailFormatter.amountLabel(record.direction, record.amountAtomic)
                } else {
                    "Unavailable"
                },
                counterpartyLabel = TransactionDetailFormatter.counterpartyLabel(record.direction),
                counterparty = if (TransactionDetailFormatter.hasKnownCounterparty(record.counterparty)) {
                    record.counterparty
                } else {
                    "Unavailable"
                },
                networkFee = if (record.direction == TransactionDirection.SENT) {
                    TransactionDetailFormatter.feeLabel(record.feeAtomic)
                } else {
                    null
                },
                dateTime = TransactionDetailFormatter.timestampLabel(record.timestampMs),
                status = item.state.name,
                confirmations = item.confirmations,
                transactionId = record.id,
                explorerUrl = ExplorerLinks.transactionUrl(record.id),
            )
        }
    }
}
''')

# Restore initial test-token request support in the same live faucet client.
f = FAUCET.read_text()
f = replace_once(
    f,
    '''    data class ChallengeReward(
''',
    '''    data class InitialGrant(
        val address: String,
        val amountSudh: Int,
        val transactionId: String,
        val status: String,
    )

    data class ChallengeReward(
''',
    "initial grant model",
)
f = replace_once(
    f,
    '''    suspend fun claimChallenge(address: String, transactionId: String): ChallengeReward {
''',
    '''    suspend fun requestInitial(address: String): InitialGrant {
        require(address.matches(Regex("^[0-9a-f]{40}$"))) { "invalid Sudharma address" }
        val dto = post(
            "v1/faucet/request",
            InitialRequest(address),
            InitialGrantDto::class.java,
        )
        return InitialGrant(dto.address, dto.amountSudh, dto.transactionId, dto.status)
    }

    suspend fun claimChallenge(address: String, transactionId: String): ChallengeReward {
''',
    "initial grant request",
)
f = replace_once(
    f,
    '''    @JsonClass(generateAdapter = false)
    data class ChallengeRequest(
''',
    '''    @JsonClass(generateAdapter = false)
    data class InitialRequest(val address: String)

    @JsonClass(generateAdapter = false)
    data class InitialGrantDto(
        val address: String,
        @Json(name = "amount_sudh") val amountSudh: Int,
        @Json(name = "transaction_id") val transactionId: String,
        val status: String,
    )

    @JsonClass(generateAdapter = false)
    data class ChallengeRequest(
''',
    "initial grant dto",
)
FAUCET.write_text(f)

# Repository exposes the request action without disturbing detailed local history.
r = REPO.read_text()
r = replace_once(
    r,
    '''    suspend fun claimChallengeReward(transactionId: String): TestnetFaucetClient.ChallengeReward =
        faucetClient().claimChallenge(account().address, transactionId)
''',
    '''    suspend fun requestInitialTestTokens(): TestnetFaucetClient.InitialGrant =
        faucetClient().requestInitial(account().address)

    suspend fun claimChallengeReward(transactionId: String): TestnetFaucetClient.ChallengeReward =
        faucetClient().claimChallenge(account().address, transactionId)
''',
    "repository initial grant action",
)
REPO.write_text(r)

# Wallet UI: visible test-token request on Home + tap-to-open full Activity details.
w = WALLET.read_text()
w = replace_once(
    w,
    '''    var recentActivity by remember { mutableStateOf<List<WalletActivityItem>>(emptyList()) }
''',
    '''    var recentActivity by remember { mutableStateOf<List<WalletActivityItem>>(emptyList()) }
    var faucetInfo by remember { mutableStateOf<TestnetFaucetClient.Info?>(null) }
    var faucetMessage by remember { mutableStateOf("") }
    var faucetLoading by remember { mutableStateOf(false) }
''',
    "home faucet state",
)
w = replace_once(
    w,
    '''    LaunchedEffect(repository.preferences.rpcUrl) { refresh() }
''',
    '''    LaunchedEffect(repository.preferences.rpcUrl) {
        refresh()
        if (repository.preferences.rpcUrl.isNotBlank()) {
            runCatching { repository.faucetInfo() }
                .onSuccess { faucetInfo = it }
                .onFailure { faucetInfo = null }
        }
    }
''',
    "home faucet load",
)
w = replace_once(
    w,
    '''        Text("Swap — Coming later", style = MaterialTheme.typography.labelSmall)

        Text("Assets", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
''',
    '''        Text("Swap — Coming later", style = MaterialTheme.typography.labelSmall)

        Card(Modifier.fillMaxWidth()) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("Sudharma Testnet Tokens", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                Text("Need test coins? Request the current testnet grant directly to this wallet. Test SUDH has no monetary value.")
                Button(
                    enabled = faucetInfo?.enabled == true && account != null && !faucetLoading,
                    onClick = {
                        faucetLoading = true
                        faucetMessage = "Requesting test SUDH…"
                        scope.launch {
                            runCatching { repository.requestInitialTestTokens() }
                                .onSuccess { grant ->
                                    faucetMessage = "${grant.amountSudh} Test SUDH request submitted. TX: ${grant.transactionId.take(12)}…"
                                    refresh()
                                }
                                .onFailure { failure ->
                                    faucetMessage = failure.message ?: "Unable to request test SUDH"
                                }
                            faucetLoading = false
                        }
                    },
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(if (faucetLoading) "Requesting…" else "Request Test SUDH") }
                if (faucetInfo?.enabled != true) {
                    Text("The protected testnet faucet is currently unavailable. This action enables automatically when the service is online.", style = MaterialTheme.typography.bodySmall)
                }
                if (faucetMessage.isNotEmpty()) Text(faucetMessage, style = MaterialTheme.typography.bodySmall)
            }
        }

        Text("Assets", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
''',
    "home faucet card",
)

activity_start = w.index('@Composable\nprivate fun ActivityScreen')
detail_row_start = w.index('@Composable\nprivate fun DetailRow', activity_start)
activity_block = '''@Composable
private fun ActivityScreen(repository: SudharmaWalletRepository, onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    var items by remember { mutableStateOf<List<WalletActivityItem>>(emptyList()) }
    var selected by remember { mutableStateOf<WalletActivityItem?>(null) }
    var message by remember { mutableStateOf("") }

    fun refresh() {
        if (repository.preferences.rpcUrl.isBlank()) {
            message = "Configure Testnet RPC in Settings first."
            return
        }
        scope.launch {
            runCatching { repository.activityHistory() }
                .onSuccess {
                    items = it
                    message = if (it.isEmpty()) "No transactions yet." else ""
                }
                .onFailure { message = it.message ?: "Unable to load activity" }
        }
    }

    LaunchedEffect(Unit) { refresh() }

    selected?.let { item ->
        TransactionDetailScreen(item = item, onBack = { selected = null })
        return
    }

    ScreenFrame("Activity", onBack) {
        TestnetBadge()
        Text("Sent and received transactions are sorted newest first. Tap any record for full transaction details.", style = MaterialTheme.typography.bodySmall)
        OutlinedButton(onClick = { refresh() }) { Text("Refresh") }
        if (message.isNotEmpty()) Text(message)
        items.forEach { item ->
            TransactionHistoryCard(item = item, onOpen = { selected = item })
        }
    }
}

@Composable
private fun TransactionHistoryCard(item: WalletActivityItem, onOpen: () -> Unit = {}) {
    val record = item.record
    Card(Modifier.fillMaxWidth().clickable(onClick = onOpen)) {
        Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text(TransactionDetailFormatter.directionLabel(record.direction), fontWeight = FontWeight.Bold)
                Text(item.state.name)
            }
            DetailRow(
                label = "Amount",
                value = if (TransactionDetailFormatter.hasKnownAmount(record)) {
                    TransactionDetailFormatter.amountLabel(record.direction, record.amountAtomic)
                } else {
                    "Unavailable"
                },
            )
            DetailRow(label = "Date & time", value = TransactionDetailFormatter.timestampLabel(record.timestampMs))
            Text("Tap to view full transaction details", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
        }
    }
}

@Composable
private fun TransactionDetailScreen(item: WalletActivityItem, onBack: () -> Unit) {
    val detail = TransactionDetailPresentation.from(item)
    val context = LocalContext.current
    ScreenFrame("Transaction Details", onBack) {
        TestnetBadge()
        DetailRow(label = "Direction", value = detail.direction)
        DetailRow(label = "Amount", value = detail.amount)
        DetailRow(label = detail.counterpartyLabel, value = detail.counterparty)
        detail.networkFee?.let { DetailRow(label = "Network fee", value = it) }
        DetailRow(label = "Network", value = "Sudharma Testnet")
        DetailRow(label = "Status", value = detail.status)
        DetailRow(label = "Date & time", value = detail.dateTime)
        if (item.state == TransactionState.CONFIRMED) {
            DetailRow(label = "Confirmations", value = detail.confirmations.toString())
        }
        TransactionReferenceActions(detail.transactionId)
        OutlinedButton(
            onClick = { context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(detail.explorerUrl))) },
            modifier = Modifier.fillMaxWidth(),
        ) { Text("View on Explorer ↗") }
    }
}

'''
w = w[:activity_start] + activity_block + w[detail_row_start:]
WALLET.write_text(w)

print("wallet 0.1.4 final activity/faucet features applied")
