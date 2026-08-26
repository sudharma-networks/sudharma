package network.sudharma.wallet

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@Composable
fun PremiumLaunchScreen(
    presentation: SplashPresentation,
    modifier: Modifier = Modifier,
) {
    val logoScale = remember { Animatable(if (presentation.animate) 0.72f else 1f) }
    val logoAlpha = remember { Animatable(if (presentation.animate) 0f else 1f) }
    val textAlpha = remember { Animatable(if (presentation.animate) 0f else 1f) }
    val haloScale = remember { Animatable(if (presentation.animate) 0.84f else 1f) }
    val haloAlpha = remember { Animatable(if (presentation.animate) 0f else 0.22f) }

    LaunchedEffect(presentation.animate) {
        if (presentation.animate) {
            coroutineScope {
                launch { logoAlpha.animateTo(1f, animationSpec = tween(420)) }
                launch { logoScale.animateTo(1f, animationSpec = tween(760)) }
                launch {
                    delay(360)
                    textAlpha.animateTo(1f, animationSpec = tween(520))
                }
                launch {
                    repeat(2) {
                        haloScale.snapTo(0.84f)
                        haloAlpha.snapTo(0.34f)
                        coroutineScope {
                            launch { haloScale.animateTo(1.18f, animationSpec = tween(presentation.haloPulseMillis.toInt())) }
                            launch { haloAlpha.animateTo(0f, animationSpec = tween(presentation.haloPulseMillis.toInt())) }
                        }
                    }
                }
            }
        }
    }

    Box(
        modifier = modifier
            .fillMaxSize()
            .background(Color(0xFF04101E)),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            Box(
                modifier = Modifier.size(190.dp),
                contentAlignment = Alignment.Center,
            ) {
                Box(
                    modifier = Modifier
                        .size(174.dp)
                        .graphicsLayer {
                            scaleX = haloScale.value
                            scaleY = haloScale.value
                            alpha = haloAlpha.value
                        }
                        .border(1.dp, Color(0xFF8CC8FF), CircleShape),
                )
                Box(
                    modifier = Modifier
                        .size(146.dp)
                        .graphicsLayer {
                            scaleX = logoScale.value
                            scaleY = logoScale.value
                            alpha = logoAlpha.value
                        }
                        .background(Color(0xFF0A1726), CircleShape)
                        .border(1.dp, Color(0x334FC3FF), CircleShape)
                        .clip(CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    Image(
                        painter = painterResource(R.drawable.sudharma_logo),
                        contentDescription = "Sudharma coin logo",
                        modifier = Modifier.size(132.dp),
                    )
                }
            }

            Spacer(Modifier.height(22.dp))
            Column(
                modifier = Modifier.graphicsLayer { alpha = textAlpha.value },
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(
                    text = presentation.title,
                    style = MaterialTheme.typography.headlineLarge,
                    fontWeight = FontWeight.Black,
                    letterSpacing = 1.4.sp,
                )
                Spacer(Modifier.height(6.dp))
                Text(
                    text = presentation.subtitle,
                    style = MaterialTheme.typography.labelLarge,
                    fontWeight = FontWeight.Bold,
                    letterSpacing = 2.2.sp,
                    color = Color(0xFF9FB5C9),
                )
            }
        }
    }
}
