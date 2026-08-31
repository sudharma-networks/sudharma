package network.sudharma.wallet

import android.graphics.Bitmap
import com.google.zxing.BarcodeFormat
import com.google.zxing.qrcode.QRCodeWriter

object QrUtils {
    fun bitmap(value: String, size: Int = 720): Bitmap {
        val matrix = QRCodeWriter().encode(value, BarcodeFormat.QR_CODE, size, size)
        val pixels = IntArray(size * size)
        for (y in 0 until size) {
            for (x in 0 until size) {
                pixels[y * size + x] = if (matrix[x, y]) 0xff000000.toInt() else 0xffffffff.toInt()
            }
        }
        return Bitmap.createBitmap(size, size, Bitmap.Config.ARGB_8888).apply {
            setPixels(pixels, 0, size, 0, 0, size, size)
        }
    }
}
