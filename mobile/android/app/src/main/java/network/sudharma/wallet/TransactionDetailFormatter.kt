package network.sudharma.wallet

import java.text.DateFormat
import java.util.Date

internal object TransactionDetailFormatter {
    fun directionLabel(direction: TransactionDirection): String = when (direction) {
        TransactionDirection.SENT -> "Sent"
        TransactionDirection.RECEIVED -> "Received"
    }

    fun counterpartyLabel(direction: TransactionDirection): String = when (direction) {
        TransactionDirection.SENT -> "Sent to"
        TransactionDirection.RECEIVED -> "Received from"
    }

    fun amountLabel(direction: TransactionDirection, amountAtomic: Long): String {
        val prefix = if (direction == TransactionDirection.SENT) "-" else "+"
        return "$prefix${formatAtomic(amountAtomic)} SUDH"
    }

    fun feeLabel(feeAtomic: Long): String? =
        if (feeAtomic > 0L) "${formatAtomic(feeAtomic)} SUDH" else null

    fun timestampLabel(timestampMs: Long): String {
        if (timestampMs <= 0L) return "Time unavailable"
        return DateFormat.getDateTimeInstance(DateFormat.MEDIUM, DateFormat.SHORT)
            .format(Date(timestampMs))
    }

    fun hasKnownCounterparty(counterparty: String): Boolean =
        counterparty != PLACEHOLDER_COUNTERPARTY

    fun hasKnownAmount(record: WalletTransactionRecord): Boolean =
        hasKnownCounterparty(record.counterparty) || record.amountAtomic > 1L || record.timestampMs > 0L
}

internal const val PLACEHOLDER_COUNTERPARTY = "0000000000000000000000000000000000000000"
