package network.sudharma.wallet

import android.Manifest
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.fragment.app.FragmentActivity
import android.content.pm.PackageManager
import android.provider.Settings
import com.journeyapps.barcodescanner.ScanContract
import com.journeyapps.barcodescanner.ScanOptions
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import network.sudharma.wallet.chain.TransactionState
import network.sudharma.wallet.chain.TransactionStatus
import network.sudharma.wallet.chain.sudharma.SudharmaPaymentUri
import network.sudharma.wallet.chain.sudharma.SudharmaRpcClient
import network.sudharma.wallet.chain.sudharma.SudharmaTransaction
import network.sudharma.wallet.recovery.RecoveryPhrase
import network.sudharma.wallet.security.BiometricGate
import network.sudharma.wallet.security.setSensitiveScreen

@Composable
fun WalletApp(repository: SudharmaWalletRepository, activity: FragmentActivity) {
    var screen by remember { mutableStateOf(WalletScreen.SPLASH) }
    var generatedPhrase by remember { mutableStateOf("") }
    val splashPresentation = remember {
        SplashPresentationPolicy.forAnimatorScale(
            Settings.Global.getFloat(
                activity.contentResolver,
                Settings.Global.ANIMATOR_DURATION_SCALE,
                1f,
            ),
        )
    }

    LaunchedEffect(Unit) {
        delay(splashPresentation.delayMillis)
        screen = WalletFlow.transition(
            screen,
            WalletFlowEvent.SplashFinished(
                walletReady = repository.walletStore.hasWallet() && repository.security.hasPin(),
            ),
        )
    }

    val sensitive = WalletPresentationPolicy.isSensitive(screen)
    DisposableEffect(sensitive) {
        activity.setSensitiveScreen(sensitive)
        onDispose { if (sensitive) activity.setSensitiveScreen(false) }
    }

    BackHandler(enabled = SystemBackNavigation.intercepts(screen)) {
        screen = SystemBackNavigation.previous(screen)
    }

    when (screen) {
        WalletScreen.SPLASH -> SplashScreen(splashPresentation)
        WalletScreen.WELCOME -> WelcomeScreen(
            onCreate = {
                generatedPhrase = repository.createNewWallet()
                screen = WalletFlow.transition(screen, WalletFlowEvent.CreateSelected)
            },
            onImport = { screen = WalletFlow.transition(screen, WalletFlowEvent.ImportSelected) },
        )
        WalletScreen.RECOVERY -> RecoveryScreen(
            phrase = generatedPhrase,
            onBack = {
                generatedPhrase = ""
                screen = WalletFlow.transition(screen, WalletFlowEvent.BackToWelcome)
            },
            onContinue = {
                screen = WalletFlow.transition(screen, WalletFlowEvent.RecoveryAcknowledged)
            },
        )
        WalletScreen.CONFIRM_RECOVERY -> ConfirmRecoveryScreen(
            phrase = generatedPhrase,
            onBack = { screen = WalletFlow.transition(screen, WalletFlowEvent.BackToRecovery) },
            onConfirmed = {
                repository.importWallet(generatedPhrase)
                repository.preferences.backupAcknowledged = true
                generatedPhrase = ""
                screen = WalletFlow.transition(screen, WalletFlowEvent.BackupVerified)
            },
        )
        WalletScreen.IMPORT -> ImportScreen(
            onBack = { screen = WalletFlow.transition(screen, WalletFlowEvent.BackToWelcome) },
            onImport = { phrase ->
                repository.importWallet(phrase)
                repository.preferences.backupAcknowledged = true
                screen = WalletFlow.transition(screen, WalletFlowEvent.ImportCompleted)
            },
        )
        WalletScreen.SET_PIN -> SetPinScreen(
            onSet = { pin ->
                repository.setPin(pin)
                screen = WalletFlow.transition(screen, WalletFlowEvent.PinCreated)
            },
        )
        WalletScreen.BIOMETRIC_SETUP -> BiometricSetupScreen(
            activity = activity,
            onDone = { enabled ->
                repository.security.biometricEnabled = enabled
                screen = WalletFlow.transition(screen, WalletFlowEvent.BiometricsFinished)
            },
        )
        WalletScreen.UNLOCK -> UnlockScreen(
            repository = repository,
            activity = activity,
            onUnlocked = { screen = WalletFlow.transition(screen, WalletFlowEvent.Unlocked) },
        )
        WalletScreen.HOME -> HomeScreen(
            repository = repository,
            onReceive = { screen = WalletScreen.RECEIVE },
            onSend = { screen = WalletScreen.SEND },
            onActivity = { screen = WalletScreen.ACTIVITY },
            onHistory = { screen = WalletScreen.HISTORY },
            onSettings = { screen = WalletScreen.SETTINGS },
        )
        WalletScreen.RECEIVE -> ReceiveScreen(repository, onBack = { screen = WalletScreen.HOME })
        WalletScreen.SEND -> SendScreen(repository, activity, onBack = { screen = WalletScreen.HOME })
        WalletScreen.ACTIVITY -> ServerActivityScreen(repository, onBack = { screen = WalletScreen.HOME })
        WalletScreen.HISTORY -> TransactionHistoryScreen(repository, onBack = { screen = WalletScreen.HOME })
        WalletScreen.SETTINGS -> SettingsScreen(
            repository = repository,
            activity = activity,
            onBack = { screen = WalletScreen.HOME },
            onBackup = { screen = WalletScreen.BACKUP },
        )
        WalletScreen.BACKUP -> BackupScreen(repository, onBack = { screen = WalletScreen.SETTINGS })
    }
}

@Composable
private fun ScreenFrame(title: String, onBack: (() -> Unit)? = null, content: @Composable () -> Unit) {
    Column(
        modifier = Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        if (onBack != null) TextButton(onClick = onBack) { Text("← Back") }
        Text(title, style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Bold)
        content()
    }
}

@Composable
private fun BrandMark(modifier: Modifier = Modifier) {
    Image(
        painter = painterResource(R.drawable.sudharma_logo),
        contentDescription = "Sudharma Network logo",
        modifier = modifier.size(96.dp),
    )
}

@Composable
private fun TestnetBadge() {
    Text(
        "TESTNET",
        modifier = Modifier.background(MaterialTheme.colorScheme.tertiaryContainer, RoundedCornerShape(20.dp)).padding(horizontal = 12.dp, vertical = 5.dp),
        style = MaterialTheme.typography.labelLarge,
        fontWeight = FontWeight.Bold,
    )
}

@Composable
private fun SplashScreen(presentation: SplashPresentation) {
    val scale = remember { Animatable(if (presentation.animate) 0.68f else 1f) }
    LaunchedEffect(presentation.animate) {
        if (presentation.animate) scale.animateTo(1f, animationSpec = tween(900))
    }
    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        BrandMark(Modifier.scale(scale.value))
        Spacer(Modifier.height(20.dp))
        Text("SUDHARMA", style = MaterialTheme.typography.headlineLarge, fontWeight = FontWeight.Black)
        Text("Sudharma Wallet", style = MaterialTheme.typography.titleMedium)
        Spacer(Modifier.height(12.dp))
        TestnetBadge()
    }
}

@Composable
private fun WelcomeScreen(onCreate: () -> Unit, onImport: () -> Unit) {
    ScreenFrame("Welcome to Sudharma Wallet") {
        BrandMark()
        Text("Your keys stay on this device. Sudharma is the first chain; the wallet architecture is ready for more chains later.")
        Button(onClick = onCreate, modifier = Modifier.fillMaxWidth()) { Text("Create New Wallet") }
        OutlinedButton(onClick = onImport, modifier = Modifier.fillMaxWidth()) { Text("Import Wallet") }
        OutlinedButton(onClick = {}, enabled = false, modifier = Modifier.fillMaxWidth()) {
            Text("Continue with Google — encrypted backup setup pending")
        }
        Text("Google will never be the owner of your wallet. Your recovery phrase remains the independent backup.", style = MaterialTheme.typography.bodySmall)
    }
}

@Composable
private fun RecoveryScreen(phrase: String, onBack: () -> Unit, onContinue: () -> Unit) {
    ScreenFrame("Back up your 12 words", onBack) {
        Text("Write these words on paper in order. Never share them with anyone.")
        Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                phrase.split(' ').forEachIndexed { index, word -> Text("${index + 1}. $word", fontWeight = FontWeight.Medium) }
            }
        }
        Text("Screenshots are blocked on this screen.", style = MaterialTheme.typography.bodySmall)
        Button(onClick = onContinue, modifier = Modifier.fillMaxWidth()) { Text("I wrote it down") }
    }
}

@Composable
private fun ConfirmRecoveryScreen(phrase: String, onBack: () -> Unit, onConfirmed: () -> Unit) {
    val words = phrase.split(' ')
    var third by remember { mutableStateOf("") }
    var seventh by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    ScreenFrame("Verify your backup", onBack) {
        Text("Enter word #3 and word #7 to prove your backup is correct.")
        OutlinedTextField(third, { third = it.trim() }, label = { Text("Word #3") }, modifier = Modifier.fillMaxWidth())
        OutlinedTextField(seventh, { seventh = it.trim() }, label = { Text("Word #7") }, modifier = Modifier.fillMaxWidth())
        if (error.isNotEmpty()) Text(error, color = MaterialTheme.colorScheme.error)
        Button(
            onClick = {
                if (words.size == 12 && third.equals(words[2], true) && seventh.equals(words[6], true)) onConfirmed()
                else error = "Those words do not match. Check your written backup."
            },
            modifier = Modifier.fillMaxWidth(),
        ) { Text("Verify backup") }
    }
}

@Composable
private fun ImportScreen(onBack: () -> Unit, onImport: (String) -> Unit) {
    var phrase by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    ScreenFrame("Import Wallet", onBack) {
        Text("Enter your 12-word Sudharma mobile recovery phrase. It is processed locally and never sent to a server.")
        OutlinedTextField(
            value = phrase,
            onValueChange = { phrase = it },
            label = { Text("12-word recovery phrase") },
            minLines = 4,
            modifier = Modifier.fillMaxWidth(),
        )
        if (error.isNotEmpty()) Text(error, color = MaterialTheme.colorScheme.error)
        Button(onClick = {
            val normalized = phrase.trim().lowercase().split(Regex("\\s+")).joinToString(" ")
            if (normalized.split(' ').size == 12 && RecoveryPhrase.validate(normalized)) onImport(normalized)
            else error = "Enter a valid 12-word recovery phrase."
        }, modifier = Modifier.fillMaxWidth()) { Text("Import securely") }
    }
}

@Composable
private fun SetPinScreen(onSet: (String) -> Unit) {
    var pin by remember { mutableStateOf("") }
    var confirm by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    ScreenFrame("Create a 6-digit PIN") {
        Text("The PIN unlocks this app. Your 12-word phrase is still the recovery backup.")
        PinField("PIN", pin) { pin = it.take(6) }
        PinField("Confirm PIN", confirm) { confirm = it.take(6) }
        if (error.isNotEmpty()) Text(error, color = MaterialTheme.colorScheme.error)
        Button(onClick = {
            when {
                !pin.matches(Regex("^[0-9]{6}$")) -> error = "PIN must contain exactly six digits."
                pin != confirm -> error = "PINs do not match."
                else -> onSet(pin)
            }
        }, modifier = Modifier.fillMaxWidth()) { Text("Set PIN") }
    }
}

@Composable
private fun PinField(label: String, value: String, onValue: (String) -> Unit) {
    OutlinedTextField(
        value = value,
        onValueChange = { if (it.all(Char::isDigit)) onValue(it) },
        label = { Text(label) },
        visualTransformation = PasswordVisualTransformation(),
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword),
        modifier = Modifier.fillMaxWidth(),
        singleLine = true,
    )
}

@Composable
private fun BiometricSetupScreen(activity: FragmentActivity, onDone: (Boolean) -> Unit) {
    var message by remember { mutableStateOf("") }
    ScreenFrame("Fingerprint / Face Unlock") {
        Text("Biometrics are optional. Your PIN remains the fallback.")
        Button(onClick = {
            BiometricGate.authenticate(activity, title = "Enable biometric unlock") { ok ->
                if (ok) onDone(true) else message = "Biometric setup was not completed."
            }
        }, modifier = Modifier.fillMaxWidth()) { Text("Enable biometrics") }
        OutlinedButton(onClick = { onDone(false) }, modifier = Modifier.fillMaxWidth()) { Text("Skip for now") }
        if (message.isNotEmpty()) Text(message)
    }
}

@Composable
private fun UnlockScreen(repository: SudharmaWalletRepository, activity: FragmentActivity, onUnlocked: () -> Unit) {
    var pin by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    ScreenFrame("Unlock Sudharma Wallet") {
        BrandMark()
        TestnetBadge()
        PinField("PIN", pin) { pin = it }
        if (error.isNotEmpty()) Text(error, color = MaterialTheme.colorScheme.error)
        Button(onClick = {
            if (repository.verifyPin(pin)) onUnlocked() else error = "Incorrect PIN"
        }, modifier = Modifier.fillMaxWidth()) { Text("Unlock") }
        if (repository.security.biometricEnabled) {
            OutlinedButton(onClick = {
                BiometricGate.authenticate(activity) { ok -> if (ok) onUnlocked() else error = "Biometric authentication cancelled." }
            }, modifier = Modifier.fillMaxWidth()) { Text("Use fingerprint / face") }
        }
    }
}

@Composable
private fun HomeScreen(
    repository: SudharmaWalletRepository,
    onReceive: () -> Unit,
    onSend: () -> Unit,
    onActivity: () -> Unit,
    onHistory: () -> Unit,
    onSettings: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    val account = remember { runCatching { repository.account() }.getOrNull() }
    var balance by remember { mutableStateOf("—") }
    var status by remember { mutableStateOf(if (repository.preferences.rpcUrl.isBlank()) "RPC not configured" else "Connecting…") }
    var recentActivity by remember { mutableStateOf<List<WalletActivityItem>>(emptyList()) }
    var faucetInfo by remember { mutableStateOf<TestnetFaucetClient.Info?>(null) }
    var faucetMessage by remember { mutableStateOf("") }
    var faucetLoading by remember { mutableStateOf(false) }

    fun refresh() {
        if (repository.preferences.rpcUrl.isBlank()) { status = "RPC not configured"; return }
        scope.launch {
            runCatching { repository.balance() }
                .onSuccess { balance = it.amount.formatted(); status = "Connected" }
                .onFailure { status = it.message ?: "Offline" }
            runCatching { repository.activityHistory() }
                .onSuccess { recentActivity = it.take(5) }
        }
    }
    LaunchedEffect(repository.preferences.rpcUrl) {
        refresh()
        if (repository.preferences.rpcUrl.isNotBlank()) {
            runCatching { repository.faucetInfo() }
                .onSuccess { faucetInfo = it }
                .onFailure { faucetInfo = null }
        }
    }

    ScreenFrame("Sudharma Wallet") {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            TestnetBadge()
            TextButton(onClick = { refresh() }) { Text("Refresh") }
        }
        Text("Portfolio", style = MaterialTheme.typography.titleMedium)
        Text("$balance SUDH", style = MaterialTheme.typography.displaySmall, fontWeight = FontWeight.Bold)
        Text(status, style = MaterialTheme.typography.bodySmall)
        account?.let { Text("${it.address.take(10)}…${it.address.takeLast(8)}", style = MaterialTheme.typography.bodySmall) }

        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Button(onClick = onSend, modifier = Modifier.weight(1f)) { Text("Send") }
            Button(onClick = onReceive, modifier = Modifier.weight(1f)) { Text("Receive") }
            OutlinedButton(onClick = {}, enabled = false, modifier = Modifier.weight(1f)) { Text("Swap") }
        }
        Text("Swap — Coming later", style = MaterialTheme.typography.labelSmall)

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
        Card(Modifier.fillMaxWidth()) {
            Row(Modifier.fillMaxWidth().padding(16.dp), horizontalArrangement = Arrangement.SpaceBetween) {
                Column { Text("SUDH", fontWeight = FontWeight.Bold); Text("Sudharma Testnet", style = MaterialTheme.typography.bodySmall) }
                Text("$balance SUDH")
            }
        }
        HorizontalDivider()
        Text("Recent history", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
        if (recentActivity.isEmpty()) {
            Text("No transactions yet. Send or receive SUDH to see history here.", style = MaterialTheme.typography.bodySmall)
        } else {
            recentActivity.forEach { item ->
                TransactionHistoryCard(item)
            }
            TextButton(onClick = onHistory) { Text("View all history") }
        }
        HorizontalDivider()
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            TextButton(onClick = onActivity) { Text("Activity") }
            TextButton(onClick = onHistory) { Text("History") }
            TextButton(onClick = onSettings) { Text("Settings") }
        }
    }
}

@Composable
private fun ReceiveScreen(repository: SudharmaWalletRepository, onBack: () -> Unit) {
    val context = LocalContext.current
    val account = remember { repository.account() }
    val uri = remember(account.address) { SudharmaPaymentUri.encode(account.address) }
    val qr = remember(uri) { QrUtils.bitmap(uri) }
    ScreenFrame("Receive SUDH", onBack) {
        TestnetBadge()
        Text("Only send Sudharma Testnet SUDH to this address.", textAlign = TextAlign.Center)
        Image(bitmap = qr.asImageBitmap(), contentDescription = "Sudharma receive QR", modifier = Modifier.fillMaxWidth())
        Text(account.address, style = MaterialTheme.typography.bodySmall)
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Button(onClick = {
                val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                clipboard.setPrimaryClip(ClipData.newPlainText("Sudharma address", account.address))
            }, modifier = Modifier.weight(1f)) { Text("Copy") }
            OutlinedButton(onClick = {
                context.startActivity(Intent.createChooser(Intent(Intent.ACTION_SEND).apply {
                    type = "text/plain"; putExtra(Intent.EXTRA_TEXT, "Sudharma Testnet address: ${account.address}")
                }, "Share Sudharma address"))
            }, modifier = Modifier.weight(1f)) { Text("Share") }
        }
    }
}

@Composable
private fun SendScreen(repository: SudharmaWalletRepository, activity: FragmentActivity, onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    var recipient by remember { mutableStateOf("") }
    var amount by remember { mutableStateOf("") }
    var challengeMode by remember { mutableStateOf(false) }
    var challengeInfo by remember { mutableStateOf<TestnetFaucetClient.Info?>(null) }
    var challengeMessage by remember { mutableStateOf("") }
    var claiming by remember { mutableStateOf(false) }
    var confirm by remember { mutableStateOf(false) }
    var pin by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    var result by remember { mutableStateOf<TransactionStatus?>(null) }
    var sending by remember { mutableStateOf(false) }

    LaunchedEffect(repository.preferences.rpcUrl) {
        runCatching { repository.faucetInfo() }
            .onSuccess { challengeInfo = it }
            .onFailure { challengeInfo = null }
    }

    val scanner = rememberLauncherForActivityResult(ScanContract()) { scan ->
        scan.contents?.let { contents ->
            val parsed = SudharmaPaymentUri.parse(contents)
            if (parsed != null) recipient = parsed else error = "QR is not a valid Sudharma Testnet address."
        }
    }
    val cameraPermission = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { granted ->
        if (granted) {
            scanner.launch(
                ScanOptions().setPrompt("Scan Sudharma address")
                    .setBeepEnabled(false)
                    .setOrientationLocked(false),
            )
        } else {
            error = "Camera permission is required to scan a QR code. You can still paste an address."
        }
    }

    fun performSend() {
        sending = true; error = ""
        scope.launch {
            runCatching { repository.send(recipient, amount, challengeMode = challengeMode) }
                .onSuccess { result = it }
                .onFailure { error = it.message ?: "Transaction failed" }
            sending = false
        }
    }

    ScreenFrame("Send SUDH", onBack) {
        TestnetBadge()
        result?.let {
            Text("Transaction accepted", style = MaterialTheme.typography.titleLarge)
            DetailRow(label = "Amount", value = "-$amount SUDH")
            DetailRow(label = "Sent to", value = recipient)
            DetailRow(label = "Status", value = it.state.name)
            TransactionReferenceActions(it.id)
            if (challengeMode) {
                Text("Challenge payment submitted. The 50 Test SUDH reward can be claimed after this transaction is confirmed.")
                Button(
                    enabled = !claiming,
                    onClick = {
                        claiming = true
                        challengeMessage = "Checking transaction confirmation…"
                        scope.launch {
                            runCatching {
                                val latest = repository.transactionStatus(it.id)
                                require(latest.state == TransactionState.CONFIRMED) {
                                    "Transaction is not confirmed yet. Try again after the next testnet block."
                                }
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
                ) {
                    Text(if (claiming) "Checking…" else "Check Confirmation & Claim ${challengeInfo?.challengeRewardSudh ?: 50} Test SUDH")
                }
                if (challengeMessage.isNotEmpty()) Text(challengeMessage, style = MaterialTheme.typography.bodySmall)
            }
            Text("Saved to Activity. Refresh Home to update confirmations.", style = MaterialTheme.typography.bodySmall)
            Button(onClick = onBack, modifier = Modifier.fillMaxWidth()) { Text("Done") }
            return@ScreenFrame
        }
        if (!confirm) {
            Card(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("Testnet Challenge", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
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
                    Text("TESTNET ONLY — NO MONETARY VALUE", style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.Bold)
                }
            }
            if (challengeMode) {
                Text("Challenge mode — official address and 25 SUDH amount are prefilled.", style = MaterialTheme.typography.bodySmall, fontWeight = FontWeight.Bold)
            }
            OutlinedTextField(recipient, { recipient = it.trim(); challengeMode = false }, label = { Text("Recipient address") }, modifier = Modifier.fillMaxWidth())
            OutlinedButton(onClick = {
                when (
                    ScannerPermissionState.next(
                        ContextCompat.checkSelfPermission(activity, Manifest.permission.CAMERA) ==
                            PackageManager.PERMISSION_GRANTED,
                    )
                ) {
                    ScannerAction.OPEN_SCANNER -> scanner.launch(
                        ScanOptions().setPrompt("Scan Sudharma address")
                            .setBeepEnabled(false)
                            .setOrientationLocked(false),
                    )
                    ScannerAction.REQUEST_PERMISSION -> cameraPermission.launch(Manifest.permission.CAMERA)
                }
            }, modifier = Modifier.fillMaxWidth()) { Text("Scan QR") }
            OutlinedTextField(
                amount,
                { amount = it; challengeMode = false },
                label = { Text("Amount (SUDH)") },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                modifier = Modifier.fillMaxWidth(),
            )
            if (error.isNotEmpty()) Text(error, color = MaterialTheme.colorScheme.error)
            Button(onClick = {
                error = ""
                runCatching {
                    require(recipient.matches(Regex("^[0-9a-f]{40}$"))) { "Invalid Sudharma address" }
                    SudharmaWalletRepository.parseCoinAmount(amount)
                }.onSuccess { confirm = true }.onFailure { error = it.message ?: "Invalid payment" }
            }, modifier = Modifier.fillMaxWidth()) { Text("Review transaction") }
        } else {
            val atomic = runCatching { SudharmaWalletRepository.parseCoinAmount(amount) }.getOrDefault(0)
            val fee = runCatching { SudharmaTransaction.calculateFee(atomic) }.getOrDefault(0)
            Card(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    if (challengeMode) Text("Testnet Challenge — Send 25 / Receive 50", fontWeight = FontWeight.Bold)
                    Text("Recipient: ${recipient.take(10)}…${recipient.takeLast(8)}")
                    Text("Amount: $amount SUDH")
                    Text("Fee: ${formatAtomic(fee)} SUDH")
                    Text("Network: Sudharma Testnet")
                }
            }
            PinField("Authorize with PIN", pin) { pin = it }
            if (error.isNotEmpty()) Text(error, color = MaterialTheme.colorScheme.error)
            Button(enabled = !sending, onClick = {
                if (!repository.verifyPin(pin)) error = "Incorrect PIN" else performSend()
            }, modifier = Modifier.fillMaxWidth()) { Text(if (sending) "Sending…" else "Authorize & Send") }
            if (repository.security.biometricEnabled) {
                OutlinedButton(enabled = !sending, onClick = {
                    BiometricGate.authenticate(activity, title = "Authorize SUDH transaction") { ok ->
                        if (ok) performSend() else error = "Biometric authorization cancelled."
                    }
                }, modifier = Modifier.fillMaxWidth()) { Text("Authorize with fingerprint / face") }
            }
            TextButton(onClick = { confirm = false; pin = "" }) { Text("Edit transaction") }
        }
    }
}

@Composable
private fun ServerActivityScreen(repository: SudharmaWalletRepository, onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    var server by remember { mutableStateOf<SudharmaRpcClient.Status?>(null) }
    var connection by remember { mutableStateOf(if (repository.preferences.rpcUrl.isBlank()) "RPC not configured" else "Checking…") }
    var serverLog by remember { mutableStateOf("No server check has completed yet.") }
    var lastRefreshMs by remember { mutableStateOf<Long?>(null) }

    fun refresh() {
        if (repository.preferences.rpcUrl.isBlank()) {
            server = null
            connection = "RPC not configured"
            serverLog = "No RPC endpoint is configured for this wallet."
            lastRefreshMs = System.currentTimeMillis()
            return
        }
        connection = "Checking…"
        scope.launch {
            runCatching { repository.serverStatus() }
                .onSuccess {
                    server = it
                    connection = "Connected"
                    serverLog = "Connected to ${it.network} RPC. Height ${it.height}, ${it.peers} peer(s), ${it.mempool} transaction(s) in mempool."
                    lastRefreshMs = System.currentTimeMillis()
                }
                .onFailure {
                    server = null
                    connection = "Offline / unavailable"
                    serverLog = it.message ?: "Unable to read server status"
                    lastRefreshMs = System.currentTimeMillis()
                }
        }
    }

    LaunchedEffect(repository.preferences.rpcUrl) { refresh() }

    ScreenFrame("Activity", onBack) {
        TestnetBadge()
        Text("Server & network activity", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
        Text("Live connection details for the Sudharma Testnet RPC used by this wallet.", style = MaterialTheme.typography.bodySmall)
        OutlinedButton(onClick = { refresh() }) { Text("Refresh server status") }
        DetailRow(label = "Connection", value = connection)
        DetailRow(label = "RPC endpoint", value = repository.preferences.rpcUrl.ifBlank { "Not configured" })
        server?.let {
            DetailRow(label = "Network", value = "${it.network} (${it.symbol})")
            DetailRow(label = "Chain height", value = it.height.toString())
            DetailRow(label = "Peers", value = it.peers.toString())
            DetailRow(label = "Mempool", value = it.mempool.toString())
            DetailRow(label = "Tip hash", value = it.tipHash.ifBlank { "—" })
            DetailRow(label = "Issued supply", value = "${formatAtomic(it.issuedSupply)} SUDH")
        }
        lastRefreshMs?.let { DetailRow(label = "Last refresh", value = TransactionDetailFormatter.timestampLabel(it)) }
        HorizontalDivider()
        Text("Server log", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
        Card(Modifier.fillMaxWidth()) {
            Text(serverLog, modifier = Modifier.padding(14.dp), style = MaterialTheme.typography.bodySmall)
        }
    }
}

@Composable
private fun TransactionHistoryScreen(repository: SudharmaWalletRepository, onBack: () -> Unit) {
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
                .onFailure { message = it.message ?: "Unable to load history" }
        }
    }

    LaunchedEffect(Unit) { refresh() }

    BackHandler(enabled = selected != null) {
        selected = null
    }

    selected?.let { item ->
        TransactionDetailScreen(item = item, onBack = { selected = null })
        return
    }

    ScreenFrame("History", onBack) {
        TestnetBadge()
        Text("Sent and received transactions are sorted newest first. Tap any record for full transaction details.", style = MaterialTheme.typography.bodySmall)
        OutlinedButton(onClick = { refresh() }) { Text("Refresh history") }
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

@Composable
private fun DetailRow(label: String, value: String) {
    Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, style = MaterialTheme.typography.bodyMedium)
    }
}

@Composable
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

@Composable
private fun SettingsScreen(
    repository: SudharmaWalletRepository,
    activity: FragmentActivity,
    onBack: () -> Unit,
    onBackup: () -> Unit,
) {
    var rpc by remember { mutableStateOf(repository.preferences.rpcUrl) }
    var message by remember { mutableStateOf("") }
    ScreenFrame("Settings", onBack) {
        Text("Network", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            TestnetBadge(); Text("Sudharma Testnet — active")
        }
        OutlinedButton(onClick = {}, enabled = false, modifier = Modifier.fillMaxWidth()) { Text("Sudharma Mainnet — unavailable until launch") }

        Text("Testnet RPC", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
        OutlinedTextField(rpc, { rpc = it }, label = { Text("https://…") }, modifier = Modifier.fillMaxWidth(), singleLine = true)
        Button(onClick = {
            runCatching { repository.preferences.rpcUrl = rpc }
                .onSuccess { message = if (rpc.isBlank()) "RPC cleared." else "Testnet RPC saved." }
                .onFailure { message = it.message ?: "Invalid RPC URL" }
        }, modifier = Modifier.fillMaxWidth()) { Text("Save RPC") }

        Text("Security", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
        OutlinedButton(onClick = onBackup, modifier = Modifier.fillMaxWidth()) { Text("Back up recovery phrase") }
        if (!repository.security.biometricEnabled) {
            OutlinedButton(onClick = {
                BiometricGate.authenticate(activity, title = "Enable biometric unlock") { ok ->
                    if (ok) { repository.security.biometricEnabled = true; message = "Biometric unlock enabled." }
                    else message = "Biometric setup was not completed."
                }
            }, modifier = Modifier.fillMaxWidth()) { Text("Enable fingerprint / face") }
        } else {
            OutlinedButton(onClick = { repository.security.biometricEnabled = false; message = "Biometric unlock disabled." }, modifier = Modifier.fillMaxWidth()) {
                Text("Disable fingerprint / face")
            }
        }

        Text("Cloud backup", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
        OutlinedButton(onClick = {}, enabled = false, modifier = Modifier.fillMaxWidth()) { Text("Google encrypted backup — OAuth setup required") }
        Text("When enabled later, only client-side encrypted backup data will be uploaded. Google login alone will not decrypt the wallet.", style = MaterialTheme.typography.bodySmall)

        Text("Universal wallet", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
        Text("Sudharma is enabled first. Bitcoin, EVM and Solana adapters are planned later without changing this wallet's core architecture.")
        if (message.isNotEmpty()) Text(message)
    }
}

@Composable
private fun BackupScreen(repository: SudharmaWalletRepository, onBack: () -> Unit) {
    var pin by remember { mutableStateOf("") }
    var phrase by remember { mutableStateOf<String?>(null) }
    var error by remember { mutableStateOf("") }
    ScreenFrame("Recovery phrase", onBack) {
        if (phrase == null) {
            Text("Enter your PIN to reveal the 12-word recovery phrase. Screenshots are blocked.")
            PinField("PIN", pin) { pin = it }
            if (error.isNotEmpty()) Text(error, color = MaterialTheme.colorScheme.error)
            Button(onClick = {
                if (repository.verifyPin(pin)) phrase = repository.walletStore.loadRecoveryPhrase()
                else error = "Incorrect PIN"
            }, modifier = Modifier.fillMaxWidth()) { Text("Reveal phrase") }
        } else {
            Card(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    phrase!!.split(' ').forEachIndexed { index, word -> Text("${index + 1}. $word") }
                }
            }
            Text("Never send these words to support, a website, Google, or another person.")
        }
    }
}
