package params

const (
	// Sudharma Network identity
	NetworkName = "Sudharma Network"
	CoinName    = "Sudharma"
	CoinSymbol  = "SUDH"

	// Permanent Sudharma Network Development Treasury.
	// Receives the 0.01% development portion of transaction fees.
	// This is a PUBLIC address only.
	DevelopmentTreasuryAddress = "16d7dc9ec0495109007860a584c7cf9055da9abf"

	// 1 SUDH = 100,000,000 base units
	CoinDecimals uint64 = 100_000_000

	// Maximum monetary supply
	// Internal identifier will be renamed from SUDH to SUDH
	// in a later controlled migration step.
	MaxSupplySUDH uint64 = 100_000_000
	MaxSupply     uint64 = MaxSupplySUDH * CoinDecimals

	// Block timing
	TargetBlockTimeSeconds uint64 = 60

	// Maximum number of currently confirmed blocks that may be
	// replaced automatically during a chain reorganization.
	// At the 60-second target this represents about two hours.
	MaxAutomaticReorgDepth uint64 = 120

	// GPU-PoW v1 activation is intentionally unarmed by default. The staged
	// testnet deployment task must replace the testnet sentinel with an explicit
	// future height only after CUDA interoperability and deployment gates pass.
	GPUV1ActivationDisabled      uint64 = ^uint64(0)
	GPUV1TestnetActivationHeight uint64 = GPUV1ActivationDisabled
	GPUV1MainnetActivationHeight uint64 = GPUV1ActivationDisabled

	// Mining subsidy
	// Internal identifier will be renamed from SUDH to SUDH
	// in a later controlled migration step.
	InitialBlockRewardSUDH uint64 = 50
	InitialBlockReward     uint64 = InitialBlockRewardSUDH * CoinDecimals

	// Reward halves every 1,000,000 blocks.
	HalvingInterval uint64 = 1_000_000

	// Transaction fee:
	// 0.10% total
	// 0.01% development
	// 0.09% miner
	TotalFeeBasisPoints       uint64 = 10
	DevelopmentFeeBasisPoints uint64 = 1
	MiningFeeBasisPoints      uint64 = 9

	// No premine
	Premine uint64 = 0
)
