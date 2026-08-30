export const PROJECT_STATUS = "PRE-MAINNET · ACTIVE DEVELOPMENT" as const;
export const PROJECT_NAME = "Sudharma Network" as const;
export const COIN_SYMBOL = "SUDH" as const;
export const SITE_URL = "https://feature-website-foundation.d2mqyt0bt8s19s.amplifyapp.com" as const;
export const REPOSITORY_URL = "https://github.com/sudharma-networks/sudharma" as const;

export const SUDH_PARAMETERS = [
  ["Maximum supply (hard cap)", "51,000,000,000 SUDH"],
  ["Decimals", "8"],
  ["Initial block reward", "50 SUDH"],
  ["Target block time", "60 seconds"],
  ["Halving interval", "1,000,000 blocks"],
  ["Premine", "0"],
  ["Total transaction fee", "0.10%"],
  ["Development portion", "0.01%"],
  ["Miner portion", "0.09%"]
] as const;
