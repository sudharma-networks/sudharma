package network.sudharma.wallet

import android.os.Bundle
import android.provider.Settings
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.fragment.app.FragmentActivity
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

class MainActivity : FragmentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val repository = SudharmaWalletRepository(applicationContext)
        startTestnetAutomation(repository)

        setContent {
            val presentation = remember {
                SplashPresentationPolicy.forAnimatorScale(
                    Settings.Global.getFloat(
                        contentResolver,
                        Settings.Global.ANIMATOR_DURATION_SCALE,
                        1f,
                    ),
                )
            }
            var showPremiumLaunch by remember { mutableStateOf(true) }

            LaunchedEffect(Unit) {
                delay(presentation.delayMillis)
                showPremiumLaunch = false
            }

            SudharmaWalletTheme {
                Surface(modifier = Modifier.fillMaxSize()) {
                    Box(modifier = Modifier.fillMaxSize()) {
                        WalletApp(repository = repository, activity = this@MainActivity)
                        if (showPremiumLaunch) {
                            PremiumLaunchScreen(presentation = presentation)
                        }
                    }
                }
            }
        }
    }

    private fun startTestnetAutomation(repository: SudharmaWalletRepository) {
        val coordinator = TestnetAutomationCoordinator(
            walletReady = repository::walletReadyForAutomation,
            faucetEnabled = { runCatching { repository.faucetInfo().enabled }.getOrDefault(false) },
            balanceAtomic = { runCatching { repository.balance().amount.atomic }.getOrNull() },
            requestInitial = { repository.requestInitialTestTokens(); Unit },
            pendingChallengeId = { repository.preferences.pendingChallengeTransactionId },
            transactionConfirmed = repository::transactionConfirmed,
            transactionFailed = repository::transactionFailed,
            claimChallenge = { transactionId -> repository.claimChallengeReward(transactionId); Unit },
            clearPendingChallenge = { repository.preferences.pendingChallengeTransactionId = null },
        )

        lifecycleScope.launch {
            repeatOnLifecycle(Lifecycle.State.STARTED) {
                while (isActive) {
                    coordinator.tick()
                    delay(AUTOMATION_TICK_MILLIS)
                }
            }
        }
    }

    companion object {
        private const val AUTOMATION_TICK_MILLIS = 5_000L
    }
}
