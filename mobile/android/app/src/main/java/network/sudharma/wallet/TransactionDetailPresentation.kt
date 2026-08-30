package network.sudharma.wallet

data class TransactionDetailPresentation(
    val direction: String,
    val amount: String,
    val counterpartyLabel: String,
    val counterparty: String,
    val networkFee: String?,
    val dateTime: String,
    val status: String,
    val confirmations: Long,
    val transactionId: String,
    val explorerUrl: String,
) {
    companion object {
        fun from(item: WalletActivityItem): TransactionDetailPresentation {
            val record = item.record
            return TransactionDetailPresentation(
                direction = TransactionDetailFormatter.directionLabel(record.direction),
                amount = if (TransactionDetailFormatter.hasKnownAmount(record)) {
                    TransactionDetailFormatter.amountLabel(record.direction, record.amountAtomic)
                } else {
                    "Unavailable"
                },
                counterpartyLabel = TransactionDetailFormatter.counterpartyLabel(record.direction),
                counterparty = if (TransactionDetailFormatter.hasKnownCounterparty(record.counterparty)) {
                    record.counterparty
                } else {
                    "Unavailable"
                },
                networkFee = if (record.direction == TransactionDirection.SENT) {
                    TransactionDetailFormatter.feeLabel(record.feeAtomic)
                } else {
                    null
                },
                dateTime = TransactionDetailFormatter.timestampLabel(record.timestampMs),
                status = item.state.name,
                confirmations = item.confirmations,
                transactionId = record.id,
                explorerUrl = ExplorerLinks.transactionUrl(record.id),
            )
        }
    }
}
