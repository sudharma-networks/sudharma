package network.sudharma.wallet.security

import android.app.Activity
import android.view.WindowManager

fun Activity.setSensitiveScreen(enabled: Boolean) {
    if (enabled) {
        window.addFlags(WindowManager.LayoutParams.FLAG_SECURE)
    } else {
        window.clearFlags(WindowManager.LayoutParams.FLAG_SECURE)
    }
}
