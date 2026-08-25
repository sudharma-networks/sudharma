package network.sudharma.wallet

enum class ScannerAction { REQUEST_PERMISSION, OPEN_SCANNER }

object ScannerPermissionState {
    fun next(cameraPermissionGranted: Boolean): ScannerAction =
        if (cameraPermissionGranted) ScannerAction.OPEN_SCANNER else ScannerAction.REQUEST_PERMISSION
}
