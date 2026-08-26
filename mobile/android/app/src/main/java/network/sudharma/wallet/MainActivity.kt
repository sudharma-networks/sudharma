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
import kotlinx.coroutines.delay

class MainActivity : FragmentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val repository = SudharmaWalletRepository(applicationContext)

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
}
