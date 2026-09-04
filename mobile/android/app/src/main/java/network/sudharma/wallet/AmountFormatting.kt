package network.sudharma.wallet

import java.math.BigDecimal

internal fun formatAtomic(value: Long): String =
    BigDecimal.valueOf(value).movePointLeft(8).setScale(8).toPlainString()
