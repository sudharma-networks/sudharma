package params

type EmissionEpoch struct {
	Issuance uint64
}

var MainnetEmissionEpochs = [40]EmissionEpoch{
	{2_101_200 * CoinDecimals}, {2_060_400 * CoinDecimals}, {2_019_600 * CoinDecimals}, {1_978_800 * CoinDecimals},
	{1_838_550 * CoinDecimals}, {1_802_850 * CoinDecimals}, {1_767_150 * CoinDecimals}, {1_731_450 * CoinDecimals},
	{1_707_225 * CoinDecimals}, {1_674_075 * CoinDecimals}, {1_640_925 * CoinDecimals}, {1_607_775 * CoinDecimals},
	{1_575_900 * CoinDecimals}, {1_545_300 * CoinDecimals}, {1_514_700 * CoinDecimals}, {1_484_100 * CoinDecimals},
	{1_444_575 * CoinDecimals}, {1_416_525 * CoinDecimals}, {1_388_475 * CoinDecimals}, {1_360_425 * CoinDecimals},
	{1_313_250 * CoinDecimals}, {1_287_750 * CoinDecimals}, {1_262_250 * CoinDecimals}, {1_236_750 * CoinDecimals},
	{1_050_600 * CoinDecimals}, {1_030_200 * CoinDecimals}, {1_009_800 * CoinDecimals}, {989_400 * CoinDecimals},
	{919_275 * CoinDecimals}, {901_425 * CoinDecimals}, {883_575 * CoinDecimals}, {865_725 * CoinDecimals},
	{656_625 * CoinDecimals}, {643_875 * CoinDecimals}, {631_125 * CoinDecimals}, {618_375 * CoinDecimals},
	{525_300 * CoinDecimals}, {515_100 * CoinDecimals}, {504_900 * CoinDecimals}, {494_700 * CoinDecimals},
}
