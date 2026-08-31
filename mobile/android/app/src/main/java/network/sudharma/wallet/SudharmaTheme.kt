package network.sudharma.wallet

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

private val SudharmaDarkColorScheme = darkColorScheme(
    primary = Color(0xFF8CC8FF),
    onPrimary = Color(0xFF001D33),
    primaryContainer = Color(0xFF123A5A),
    onPrimaryContainer = Color(0xFFD7ECFF),
    secondary = Color(0xFFE2C778),
    onSecondary = Color(0xFF2A2105),
    secondaryContainer = Color(0xFF4A3D13),
    onSecondaryContainer = Color(0xFFFFECA8),
    tertiary = Color(0xFF9AD7C7),
    tertiaryContainer = Color(0xFF153E38),
    background = Color(0xFF06111F),
    onBackground = Color(0xFFEAF2FA),
    surface = Color(0xFF0A1726),
    onSurface = Color(0xFFEAF2FA),
    surfaceVariant = Color(0xFF142437),
    onSurfaceVariant = Color(0xFFC2CFDD),
)

@Composable
fun SudharmaWalletTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = SudharmaDarkColorScheme,
        content = content,
    )
}
