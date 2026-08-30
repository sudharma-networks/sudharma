package network.sudharma.wallet

import android.Manifest
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.fragment.app.FragmentActivity
import androidx.core.content.ContextCompat
import android.content.pm.PackageManager
import android.provider.Settings
import com.journeyapps.barcodescanner.ScanContract
import com.journeyapps.barcodescanner.ScanOptions
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import network.sudharma.wallet.chain.TransactionState
import network.sudharma.wallet.chain.TransactionStatus
import network.sudharma.wallet.chain.sudharma.SudharmaPaymentUri
import network.sudharma.wallet.chain.sudharma.SudharmaTransaction
import network.sudharma.wallet.recovery.RecoveryPhrase
import network.sudharma.wallet.security.BiometricGate
import network.sudharma.wallet.security.setSensitiveScreen
import java.math.BigDecimal

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

    // This private treasury build blocks screenshots and screen recording on every screen.
    val sensitive = true
    DisposableEffect(sensitive) {
        activity.setSensitiveScreen(sensitive)
        onDispose { if (sensitive) activity.setSensitiveScreen(false) }
    }

    when (screen) {
        WalletScreen.SPLASH -> SplashScreen(splashPresentation)
        WalletScreen.WELCOME -> WelcomeScreen(
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
                screen = WalletFlow.transition(screen, WalletFlowEvent.BackupVerified)
            },
        )
        WalletScreen.IMPORT -> ImportScreen(
            onBack = { screen = WalletFlow.transition(screen, WalletFlowEvent.BackToWelcome) },
            onTreasuryImport = { jsonBytes, password ->
                repository.importTreasuryWallet(jsonBytes, password)
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
            onSettings = { screen = WalletScreen.SETTINGS },
        )
        WalletScreen.RECEIVE -> ReceiveScreen(repository, onBack = { screen = WalletScreen.HOME })
        WalletScreen.SEND -> SendScreen(repository, activity, onBack = { screen = WalletScreen.HOME })
        WalletScreen.ACTIVITY -> ActivityScreen(repository, onBack = { screen = WalletScreen.HOME })
        WalletScreen.SETTINGS -> SettingsScreen(
            repository = repository,
            activity = activity,
            onBack = { screen = WalletScreen.HOME },
            onBackup = { screen = WalletScreen.BACKUP },
            onRemoved = { screen = WalletScreen.WELCOME },
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
    Box(
        modifier = modifier.size(96.dp).background(MaterialTheme.colorScheme.primaryContainer, CircleShape),
        contentAlignment = Alignment.Center,
    ) {
        Text("S", style = MaterialTheme.typography.displaySmall, fontWeight = FontWeight.Black)
    }
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
private fun WelcomeScreen(onImport: () -> Unit) {
    ScreenFrame("Sudharma Treasury — Private") {
        BrandMark()
        Text("Private testnet treasury controller. Import the existing encrypted Sudharma JSON wallet locally; no key is included in this APK.")
        Button(onClick = onImport, modifier = Modifier.fillMaxWidth()) { Text("Import Treasury JSON") }
        Text("The selected file and password stay on this device. Keep the original JSON and raw private key offline.", style = MaterialTheme.typography.bodySmall)
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
private fun ImportScreen(onBack: () -> Unit, onTreasuryImport: (ByteArray, CharArray) -> Unit) {
    val context = LocalContext.current
    var walletBytes by remember { mutableStateOf<ByteArray?>(null) }
    var selectedFile by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    val picker = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { uri ->
        error = ""
        walletBytes?.fill(0)
        walletBytes = null
        selectedFile = ""
        if (uri != null) {
            runCatching {
                context.contentResolver.openInputStream(uri)?.use { it.readBytes() }
                    ?: throw IllegalArgumentException("Unable to read selected wallet file.")
            }.onSuccess {
                walletBytes = it
                selectedFile = uri.lastPathSegment?.substringAfterLast('/') ?: "Selected treasury JSON"
            }.onFailure {
                error = it.message ?: "Unable to read selected wallet file."
            }
        }
    }
    ScreenFrame("Import Treasury Wallet", onBack) {
        Text("Select the encrypted Sudharma .json wallet. The app will unlock it locally and accept it only if it derives the permanent treasury address.")
        OutlinedButton(
            onClick = { picker.launch(arrayOf("application/json", "text/plain", "application/octet-stream")) },
            modifier = Modifier.fillMaxWidth(),
        ) { Text(if (selectedFile.isEmpty()) "Select encrypted JSON" else selectedFile) }
        OutlinedTextField(
            value = password,
            onValueChange = { password = it },
            label = { Text("JSON wallet password") },
            visualTransformation = PasswordVisualTransformation(),
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
        )
        if (error.isNotEmpty()) Text(error, color = MaterialTheme.colorScheme.error)
        Button(onClick = {
            val bytes = walletBytes
            if (bytes == null) {
                error = "Select the encrypted treasury JSON file."
            } else if (password.isEmpty()) {
                error = "Enter the JSON wallet password."
            } else {
                runCatching { onTreasuryImport(bytes, password.toCharArray()) }
                    .onSuccess {
                        password = ""
                        walletBytes = null
                    }
                    .onFailure {
                        password = ""
                        walletBytes = null
                        selectedFile = ""
                        error = it.message ?: "Treasury import failed."
                    }
            }
        }, modifier = Modifier.fillMaxWidth()) { Text("Verify & Import Treasury") }
        Text("Required address: " + SudharmaWalletRepository.DEVELOPMENT_TREASURY_ADDRESS, style = MaterialTheme.typography.bodySmall)
    }
}

@Composable
private fun SetPinScreen(onSet: (String) -> Unit) {
    var pin by remember { mutableStateOf("") }
    var confirm by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    ScreenFrame("Create a 6-digit PIN") {
        Text("The PIN unlocks this private treasury app. Your original encrypted JSON and raw key remain the offline recovery backups.")
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
    onSettings: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    val account = remember { runCatching { repository.account() }.getOrNull() }
    var balance by remember { mutableStateOf("—") }
    var status by remember { mutableStateOf("Connecting…") }
    var faucetMessage by remember { mutableStateOf("") }
    var faucetInfo by remember { mutableStateOf<TestnetFaucetClient.Info?>(null) }
    var faucetLoading by remember { mutableStateOf(false) }

    fun refresh() {
        scope.launch {
            runCatching { repository.balance() }
                .onSuccess { balance = it.amount.formatted(); status = "Connected" }
                .onFailure { status = it.message ?: "Offline" }
        }
    }

    fun refreshFaucet() {
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

    ScreenFrame(if (repository.isTreasuryWallet()) "Sudharma Treasury — Private" else "Sudharma Wallet") {
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

        if (!repository.isTreasuryWallet()) Card(Modifier.fillMaxWidth()) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("Sudharma Testnet Faucet", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
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
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            TextButton(onClick = onActivity) { Text("Activity") }
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
    var confirm by remember { mutableStateOf(false) }
    var pin by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    var result by remember { mutableStateOf<TransactionStatus?>(null) }
    var sending by remember { mutableStateOf(false) }
    var challengeInfo by remember { mutableStateOf<TestnetFaucetClient.Info?>(null) }
    var challengeMessage by remember { mutableStateOf("") }
    var claiming by remember { mutableStateOf(false) }

    LaunchedEffect(repository.preferences.rpcUrl) {
        if (!repository.isTreasuryWallet()) {
            runCatching { repository.faucetInfo() }
                .onSuccess { challengeInfo = it }
                .onFailure { challengeInfo = null }
        }
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
            runCatching { repository.send(recipient, amount) }
                .onSuccess { result = it }
                .onFailure { error = it.message ?: "Transaction failed" }
            sending = false
        }
    }

    ScreenFrame("Send SUDH", onBack) {
        TestnetBadge()
        result?.let {
            Text("Transaction accepted", style = MaterialTheme.typography.titleLarge)
            Text(it.id, style = MaterialTheme.typography.bodySmall)
            Text("Status: ${it.state}")
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
        }
        if (!confirm) {
            if (!repository.isTreasuryWallet()) Card(Modifier.fillMaxWidth()) {
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
                Text("Challenge mode", fontWeight = FontWeight.Bold)
                OutlinedTextField(recipient, {}, readOnly = true, label = { Text("Official challenge address") }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(amount, {}, readOnly = true, label = { Text("Challenge amount (SUDH)") }, modifier = Modifier.fillMaxWidth())
                TextButton(onClick = { challengeMode = false; recipient = ""; amount = "" }) { Text("Switch to normal Send") }
            } else {
                OutlinedTextField(recipient, { recipient = it.trim() }, label = { Text("Recipient address") }, modifier = Modifier.fillMaxWidth())
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
                    { amount = it },
                    label = { Text("Amount (SUDH)") },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                    modifier = Modifier.fillMaxWidth(),
                )
            }
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
                    if (challengeMode) Text("Testnet Challenge", fontWeight = FontWeight.Bold)
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
private fun ActivityScreen(repository: SudharmaWalletRepository, onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    var statuses by remember { mutableStateOf<List<TransactionStatus>>(emptyList()) }
    var message by remember { mutableStateOf("") }
    fun refresh() {
        scope.launch {
            runCatching { repository.transactionStatuses() }
                .onSuccess { statuses = it; message = if (it.isEmpty()) "No submitted transactions yet." else "" }
                .onFailure { message = it.message ?: "Unable to load activity" }
        }
    }
    LaunchedEffect(Unit) { refresh() }
    ScreenFrame("Activity", onBack) {
        TestnetBadge()
        OutlinedButton(onClick = { refresh() }) { Text("Refresh") }
        if (message.isNotEmpty()) Text(message)
        statuses.forEach { tx ->
            Card(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(14.dp)) {
                    Text(tx.id.take(14) + "…" + tx.id.takeLast(8), fontWeight = FontWeight.Medium)
                    Text(tx.state.name)
                    if (tx.state == TransactionState.CONFIRMED) Text("Confirmations: ${tx.confirmations}")
                }
            }
        }
    }
}

@Composable
private fun SettingsScreen(
    repository: SudharmaWalletRepository,
    activity: FragmentActivity,
    onBack: () -> Unit,
    onBackup: () -> Unit,
    onRemoved: () -> Unit,
) {
    var rpc by remember { mutableStateOf(repository.preferences.rpcUrl) }
    var message by remember { mutableStateOf("") }
    var removeTreasury by remember { mutableStateOf(false) }
    var removePin by remember { mutableStateOf("") }
    ScreenFrame("Settings", onBack) {
        Text("Network", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            TestnetBadge(); Text("Sudharma Testnet — active")
        }
        OutlinedButton(onClick = {}, enabled = false, modifier = Modifier.fillMaxWidth()) { Text("Sudharma Mainnet — unavailable until launch") }

        Text("Testnet connection", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
        Text("Automatically managed by Sudharma Wallet")
        Text(repository.preferences.rpcUrl, style = MaterialTheme.typography.bodySmall)
        if (BuildConfig.DEBUG) {
            OutlinedTextField(rpc, { rpc = it }, label = { Text("Debug RPC override") }, modifier = Modifier.fillMaxWidth(), singleLine = true)
            Button(onClick = {
                runCatching { repository.preferences.rpcUrl = rpc }
                    .onSuccess {
                        rpc = repository.preferences.rpcUrl
                        message = "Debug RPC override saved."
                    }
                    .onFailure { message = it.message ?: "Invalid RPC URL" }
            }, modifier = Modifier.fillMaxWidth()) { Text("Save debug override") }
        }

        Text("Security", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
        if (repository.isTreasuryWallet()) {
            Text("Treasury recovery stays offline in your original encrypted JSON and raw private-key backup.", style = MaterialTheme.typography.bodySmall)
            OutlinedButton(onClick = { removeTreasury = true }, modifier = Modifier.fillMaxWidth()) {
                Text("Remove treasury wallet from this device")
            }
            if (removeTreasury) {
                Text("Enter your PIN to erase the app-held treasury key. Your original JSON and raw-key backups are not changed.")
                PinField("PIN", removePin) { removePin = it }
                Button(
                    onClick = {
                        if (repository.verifyPin(removePin)) {
                            repository.resetWallet()
                            onRemoved()
                        } else {
                            message = "Incorrect PIN"
                        }
                    },
                    modifier = Modifier.fillMaxWidth(),
                ) { Text("Confirm removal") }
                TextButton(onClick = { removeTreasury = false; removePin = "" }) { Text("Cancel") }
            }
        } else {
            OutlinedButton(onClick = onBackup, modifier = Modifier.fillMaxWidth()) { Text("Back up recovery phrase") }
        }
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

        if (!repository.isTreasuryWallet()) {
            Text("Cloud backup", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
            OutlinedButton(onClick = {}, enabled = false, modifier = Modifier.fillMaxWidth()) { Text("Google encrypted backup — OAuth setup required") }
            Text("When enabled later, only client-side encrypted backup data will be uploaded. Google login alone will not decrypt the wallet.", style = MaterialTheme.typography.bodySmall)
        }

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

private fun formatAtomic(value: Long): String = BigDecimal.valueOf(value).movePointLeft(8).setScale(8).toPlainString()
