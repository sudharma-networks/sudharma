package params

// Transaction and mempool resource limits guard consensus and node memory/CPU
// against disproportionately cheap spam. Values are deliberately conservative
// for public testnet and can be reviewed before mainnet freeze.
const (
	// MinTransferAmount is the smallest transfer that may pay a non-zero fee at
	// the configured 0.10% basis-point rate. Smaller transfers are rejected as
	// dust because they create state work without economic cost.
	MinTransferAmount uint64 = 1000

	MaxTransactionIDLength      = 64
	MaxTransactionPublicKeySize = 65
	MaxTransactionSignatureSize = 64

	MaxMempoolTransactions = 4096
	MaxMempoolBytes        = 4 * 1024 * 1024

	MaxBlockTransactions     = 1000
	MaxBlockTransactionBytes = 1024 * 1024
)
