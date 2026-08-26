from pathlib import Path

path = Path('mobile/android/app/src/main/java/network/sudharma/wallet/WalletApp.kt')
text = path.read_text()

old = '''    var faucetMessage by remember { mutableStateOf("") }

    fun refresh() {
        if (repository.preferences.rpcUrl.isBlank()) { status = "RPC not configured"; return }
        scope.launch {
            runCatching { repository.balance() }
                .onSuccess { balance = it.amount.formatted(); status = "Connected" }
                .onFailure { status = it.message ?: "Offline" }
        }
    }
    LaunchedEffect(repository.preferences.rpcUrl) { refresh() }
'''
new = '''    var faucetMessage by remember { mutableStateOf("") }
    var faucetInfo by remember { mutableStateOf<TestnetFaucetClient.Info?>(null) }
    var faucetLoading by remember { mutableStateOf(false) }

    fun refresh() {
        if (repository.preferences.rpcUrl.isBlank()) { status = "RPC not configured"; return }
        scope.launch {
            runCatching { repository.balance() }
                .onSuccess { balance = it.amount.formatted(); status = "Connected" }
                .onFailure { status = it.message ?: "Offline" }
        }
    }

    fun refreshFaucet() {
        if (repository.preferences.rpcUrl.isBlank()) return
        scope.launch {
            runCatching { repository.faucetInfo() }
                .onSuccess { faucetInfo = it }
                .onFailure { faucetInfo = null }
        }
    }

    LaunchedEffect(repository.preferences.rpcUrl) {
        refresh()
        refreshFaucet()
    }
'''
assert old in text, 'home state block not found'
text = text.replace(old, new, 1)

old = '''                Text("Sudharma Testnet Faucet", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                Text("New testers can request one initial ${TestnetChallengeConfig.INITIAL_GRANT_SUDH} SUDH test grant. Test SUDH has no monetary value.")
                Button(
                    enabled = TestnetChallengeConfig.faucetEnabled,
                    onClick = { faucetMessage = "Submitting test-token request…" },
                    modifier = Modifier.fillMaxWidth(),
                ) { Text("Request ${TestnetChallengeConfig.INITIAL_GRANT_SUDH} Test SUDH") }
                if (!TestnetChallengeConfig.faucetEnabled) {
                    Text("Faucet activation is in progress. The button will activate after the protected faucet backend is deployed.", style = MaterialTheme.typography.bodySmall)
                }
                if (faucetMessage.isNotEmpty()) Text(faucetMessage, style = MaterialTheme.typography.bodySmall)
'''
new = '''                Text("Sudharma Testnet Faucet", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                Text("New testers can request one initial ${faucetInfo?.initialGrantSudh ?: 100} SUDH test grant. Test SUDH has no monetary value.")
                Button(
                    enabled = faucetInfo?.enabled == true && account != null && !faucetLoading,
                    onClick = {
                        faucetLoading = true
                        faucetMessage = "Submitting test-token request…"
                        scope.launch {
                            runCatching { repository.requestInitialTestTokens() }
                                .onSuccess {
                                    faucetMessage = "${it.amountSudh} Test SUDH submitted. Transaction: ${it.transactionId.take(12)}…"
                                    refresh()
                                }
                                .onFailure { faucetMessage = it.message ?: "Unable to request test tokens" }
                            faucetLoading = false
                        }
                    },
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(if (faucetLoading) "Requesting…" else "Request ${faucetInfo?.initialGrantSudh ?: 100} Test SUDH") }
                if (faucetInfo?.enabled != true) {
                    Text("Faucet is currently unavailable. The wallet will enable this automatically when the protected testnet faucet is online.", style = MaterialTheme.typography.bodySmall)
                }
                if (faucetMessage.isNotEmpty()) Text(faucetMessage, style = MaterialTheme.typography.bodySmall)
'''
assert old in text, 'home faucet card block not found'
text = text.replace(old, new, 1)

old = '''    var result by remember { mutableStateOf<TransactionStatus?>(null) }
    var sending by remember { mutableStateOf(false) }

    val scanner = rememberLauncherForActivityResult(ScanContract()) { scan ->
'''
new = '''    var result by remember { mutableStateOf<TransactionStatus?>(null) }
    var sending by remember { mutableStateOf(false) }
    var challengeInfo by remember { mutableStateOf<TestnetFaucetClient.Info?>(null) }
    var challengeMessage by remember { mutableStateOf("") }
    var claiming by remember { mutableStateOf(false) }

    LaunchedEffect(repository.preferences.rpcUrl) {
        runCatching { repository.faucetInfo() }
            .onSuccess { challengeInfo = it }
            .onFailure { challengeInfo = null }
    }

    val scanner = rememberLauncherForActivityResult(ScanContract()) { scan ->
'''
assert old in text, 'send state block not found'
text = text.replace(old, new, 1)

old = '''            if (challengeMode) Text("Challenge payment submitted. A reward is issued only after the backend verifies confirmation and eligibility.")
            Button(onClick = onBack, modifier = Modifier.fillMaxWidth()) { Text("Done") }
            return@ScreenFrame
'''
new = '''            if (challengeMode) {
                Text("Challenge payment submitted. The 50 Test SUDH reward can be claimed after this transaction is confirmed.")
                Button(
                    enabled = !claiming,
                    onClick = {
                        claiming = true
                        challengeMessage = "Checking transaction confirmation…"
                        scope.launch {
                            runCatching {
                                val latest = repository.transactionStatus(it.id)
                                require(latest.state == TransactionState.CONFIRMED) { "Transaction is not confirmed yet. Try again after the next testnet block." }
                                repository.claimChallengeReward(it.id)
                            }.onSuccess { reward ->
                                challengeMessage = "Round ${reward.round} complete — ${reward.rewardSudh} Test SUDH reward submitted."
                            }.onFailure { failure ->
                                challengeMessage = failure.message ?: "Unable to claim challenge reward"
                            }
                            claiming = false
                        }
                    },
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(if (claiming) "Checking…" else "Check Confirmation & Claim ${challengeInfo?.challengeRewardSudh ?: 50} Test SUDH") }
                if (challengeMessage.isNotEmpty()) Text(challengeMessage, style = MaterialTheme.typography.bodySmall)
            }
            Button(onClick = onBack, modifier = Modifier.fillMaxWidth()) { Text("Done") }
            return@ScreenFrame
'''
assert old in text, 'send result block not found'
text = text.replace(old, new, 1)

old = '''                    Text("Testnet Challenge", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    Text("Send ${TestnetChallengeConfig.CHALLENGE_SEND_SUDH} test SUDH to the official challenge address. After verification, eligible testers receive ${TestnetChallengeConfig.CHALLENGE_REWARD_SUDH} test SUDH back. Up to ${TestnetChallengeConfig.MAX_ROUNDS} rounds with a ${TestnetChallengeConfig.COOLDOWN_HOURS}-hour wait between successful rounds.")
                    Button(
                        enabled = TestnetChallengeConfig.challengeDepositAddress != null,
                        onClick = {
                            challengeMode = true
                            recipient = TestnetChallengeConfig.challengeDepositAddress.orEmpty()
                            amount = TestnetChallengeConfig.CHALLENGE_SEND_SUDH
                            error = ""
                        },
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("Use 25 → 50 Testnet Challenge") }
                    if (TestnetChallengeConfig.challengeDepositAddress == null) {
                        Text("Official challenge address is being provisioned. Challenge sending remains locked until that wallet is funded and published.", style = MaterialTheme.typography.bodySmall)
                    }
'''
new = '''                    Text("Testnet Challenge", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    Text("Send ${challengeInfo?.challengeSendSudh ?: 25} test SUDH to the official challenge address. After confirmation, eligible testers receive ${challengeInfo?.challengeRewardSudh ?: 50} test SUDH back. Up to ${challengeInfo?.maxRounds ?: 5} rounds with a ${challengeInfo?.cooldownHours ?: 24}-hour wait between successful rounds.")
                    Button(
                        enabled = challengeInfo?.enabled == true && challengeInfo?.challengeAddress?.matches(Regex("^[0-9a-f]{40}$")) == true,
                        onClick = {
                            challengeMode = true
                            recipient = challengeInfo?.challengeAddress.orEmpty()
                            amount = (challengeInfo?.challengeSendSudh ?: 25).toString()
                            error = ""
                        },
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("Use 25 → 50 Testnet Challenge") }
                    if (challengeInfo?.enabled != true) {
                        Text("Official challenge service is currently unavailable. It will enable automatically when the protected faucet is online.", style = MaterialTheme.typography.bodySmall)
                    }
'''
assert old in text, 'challenge card block not found'
text = text.replace(old, new, 1)

path.write_text(text)
print('WalletApp.kt faucet UI patch applied')
