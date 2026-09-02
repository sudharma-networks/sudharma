export const PROJECT_STATUS = "PRE-MAINNET · ACTIVE DEVELOPMENT" as const;
export const PROJECT_NAME = "Sudharma Network" as const;
export const COIN_SYMBOL = "SUDH" as const;
export const SITE_URL = "https://feature-website-foundation.d2mqyt0bt8s19s.amplifyapp.com" as const;
export const REPOSITORY_URL = "https://github.com/sudharma-networks/sudharma" as const;

export const SUDH_PARAMETERS = [
  ["Public testnet maximum supply (legacy hard cap)", "51,000,000,000 SUDH"],
  ["Mainnet maximum supply (final monetary policy)", "51,000,000 SUDH"],
  ["Decimals", "8"],
  ["Target block time", "60 seconds"],
  ["Mainnet subsidy-bearing blocks", "5,259,600"],
  ["Mainnet emission epochs", "40 quarterly epochs"],
  ["Mainnet nominal subsidy period", "~10 target years"],
  ["Mainnet subsidy after height 5,259,600", "0 SUDH"],
  ["Public testnet initial block reward", "50 SUDH"],
  ["Public testnet halving interval", "1,000,000 blocks"],
  ["Premine", "0"],
  ["Total transaction fee", "0.10%"],
  ["Development portion", "0.01%"],
  ["Miner portion", "0.09%"]
] as const;

export const MAINNET_EMISSION_ROADMAP = [
  ["Year 1", "16%", "8,160,000 SUDH", "8,160,000 SUDH"],
  ["Year 2", "14%", "7,140,000 SUDH", "15,300,000 SUDH"],
  ["Year 3", "13%", "6,630,000 SUDH", "21,930,000 SUDH"],
  ["Year 4", "12%", "6,120,000 SUDH", "28,050,000 SUDH"],
  ["Year 5", "11%", "5,610,000 SUDH", "33,660,000 SUDH"],
  ["Year 6", "10%", "5,100,000 SUDH", "38,760,000 SUDH"],
  ["Year 7", "8%", "4,080,000 SUDH", "42,840,000 SUDH"],
  ["Year 8", "7%", "3,570,000 SUDH", "46,410,000 SUDH"],
  ["Year 9", "5%", "2,550,000 SUDH", "48,960,000 SUDH"],
  ["Year 10", "4%", "2,040,000 SUDH", "51,000,000 SUDH"]
] as const;
