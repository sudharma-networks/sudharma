package network.sudharma.wallet

object SystemBackNavigation {
    fun intercepts(screen: WalletScreen): Boolean = when (screen) {
        WalletScreen.RECOVERY,
        WalletScreen.CONFIRM_RECOVERY,
        WalletScreen.IMPORT,
        WalletScreen.RECEIVE,
        WalletScreen.SEND,
        WalletScreen.ACTIVITY,
        WalletScreen.SETTINGS,
        WalletScreen.BACKUP,
        -> true

        else -> false
    }

    fun previous(screen: WalletScreen): WalletScreen = when (screen) {
        WalletScreen.RECOVERY -> WalletScreen.WELCOME
        WalletScreen.CONFIRM_RECOVERY -> WalletScreen.RECOVERY
        WalletScreen.IMPORT -> WalletScreen.WELCOME
        WalletScreen.RECEIVE,
        WalletScreen.SEND,
        WalletScreen.ACTIVITY,
        WalletScreen.SETTINGS,
        -> WalletScreen.HOME

        WalletScreen.BACKUP -> WalletScreen.SETTINGS
        else -> screen
    }
}
