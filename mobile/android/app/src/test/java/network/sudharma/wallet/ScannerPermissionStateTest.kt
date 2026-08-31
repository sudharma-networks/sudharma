package network.sudharma.wallet

import org.junit.Assert.assertEquals
import org.junit.Test

class ScannerPermissionStateTest {
    @Test
    fun scannerOnlyOpensAfterCameraPermissionIsGranted() {
        assertEquals(
            ScannerAction.REQUEST_PERMISSION,
            ScannerPermissionState.next(cameraPermissionGranted = false),
        )
        assertEquals(
            ScannerAction.OPEN_SCANNER,
            ScannerPermissionState.next(cameraPermissionGranted = true),
        )
    }
}
