package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Test

class WalletFlowTest {
    @Test
    fun newWalletOnboardingRequiresBackupPinAndBiometricDecision() {
        var screen = WalletScreen.SPLASH
        screen = WalletFlow.transition(screen, WalletFlowEvent.SplashFinished(walletReady = false))
        screen = WalletFlow.transition(screen, WalletFlowEvent.CreateSelected)
        screen = WalletFlow.transition(screen, WalletFlowEvent.RecoveryAcknowledged)
        screen = WalletFlow.transition(screen, WalletFlowEvent.BackupVerified)
        screen = WalletFlow.transition(screen, WalletFlowEvent.PinCreated)
        screen = WalletFlow.transition(screen, WalletFlowEvent.BiometricsFinished)

        assertEquals(WalletScreen.HOME, screen)
    }

    @Test
    fun importedWalletAlsoRequiresPinBeforeHome() {
        var screen = WalletFlow.transition(
            WalletScreen.SPLASH,
            WalletFlowEvent.SplashFinished(walletReady = false),
        )
        screen = WalletFlow.transition(screen, WalletFlowEvent.ImportSelected)
        screen = WalletFlow.transition(screen, WalletFlowEvent.ImportCompleted)
        screen = WalletFlow.transition(screen, WalletFlowEvent.PinCreated)
        screen = WalletFlow.transition(screen, WalletFlowEvent.BiometricsFinished)

        assertEquals(WalletScreen.HOME, screen)
    }

    @Test
    fun existingWalletStartsLockedAndRequiresUnlock() {
        var screen = WalletFlow.transition(
            WalletScreen.SPLASH,
            WalletFlowEvent.SplashFinished(walletReady = true),
        )
        assertEquals(WalletScreen.UNLOCK, screen)

        screen = WalletFlow.transition(screen, WalletFlowEvent.Unlocked)
        assertEquals(WalletScreen.HOME, screen)
    }

    @Test(expected = IllegalArgumentException::class)
    fun invalidTransitionIsRejected() {
        WalletFlow.transition(WalletScreen.WELCOME, WalletFlowEvent.Unlocked)
    }
}
