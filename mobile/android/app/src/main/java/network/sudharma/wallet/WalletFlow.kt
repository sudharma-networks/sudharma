package network.sudharma.wallet

enum class WalletScreen {
    SPLASH,
    WELCOME,
    RECOVERY,
    CONFIRM_RECOVERY,
    IMPORT,
    SET_PIN,
    BIOMETRIC_SETUP,
    UNLOCK,
    HOME,
    RECEIVE,
    SEND,
    ACTIVITY,
    SETTINGS,
    BACKUP,
}

sealed interface WalletFlowEvent {
    data class SplashFinished(val walletReady: Boolean) : WalletFlowEvent
    data object CreateSelected : WalletFlowEvent
    data object ImportSelected : WalletFlowEvent
    data object RecoveryAcknowledged : WalletFlowEvent
    data object BackupVerified : WalletFlowEvent
    data object ImportCompleted : WalletFlowEvent
    data object PinCreated : WalletFlowEvent
    data object BiometricsFinished : WalletFlowEvent
    data object Unlocked : WalletFlowEvent
    data object BackToWelcome : WalletFlowEvent
    data object BackToRecovery : WalletFlowEvent
}

object WalletFlow {
    fun transition(screen: WalletScreen, event: WalletFlowEvent): WalletScreen = when (screen to event) {
        WalletScreen.SPLASH to WalletFlowEvent.SplashFinished(walletReady = false) -> WalletScreen.WELCOME
        WalletScreen.SPLASH to WalletFlowEvent.SplashFinished(walletReady = true) -> WalletScreen.UNLOCK
        WalletScreen.WELCOME to WalletFlowEvent.CreateSelected -> WalletScreen.RECOVERY
        WalletScreen.WELCOME to WalletFlowEvent.ImportSelected -> WalletScreen.IMPORT
        WalletScreen.RECOVERY to WalletFlowEvent.RecoveryAcknowledged -> WalletScreen.CONFIRM_RECOVERY
        WalletScreen.RECOVERY to WalletFlowEvent.BackToWelcome -> WalletScreen.WELCOME
        WalletScreen.CONFIRM_RECOVERY to WalletFlowEvent.BackupVerified -> WalletScreen.SET_PIN
        WalletScreen.CONFIRM_RECOVERY to WalletFlowEvent.BackToRecovery -> WalletScreen.RECOVERY
        WalletScreen.IMPORT to WalletFlowEvent.ImportCompleted -> WalletScreen.SET_PIN
        WalletScreen.IMPORT to WalletFlowEvent.BackToWelcome -> WalletScreen.WELCOME
        WalletScreen.SET_PIN to WalletFlowEvent.PinCreated -> WalletScreen.BIOMETRIC_SETUP
        WalletScreen.BIOMETRIC_SETUP to WalletFlowEvent.BiometricsFinished -> WalletScreen.HOME
        WalletScreen.UNLOCK to WalletFlowEvent.Unlocked -> WalletScreen.HOME
        else -> throw IllegalArgumentException("Invalid wallet transition: $screen + $event")
    }
}
