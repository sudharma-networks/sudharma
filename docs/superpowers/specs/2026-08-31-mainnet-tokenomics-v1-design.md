# Sudharma Mainnet Tokenomics v1 — Consensus Design

Date: 2026-08-31
Status: Approved design; implementation not yet authorized by this document

## 1. Locked economic policy

- Maximum monetary supply: **51,000,000.00000000 SUDH**.
- Base unit: 1 SUDH = 100,000,000 base units.
- Premine: **0**.
- Target block time: **60 seconds**.
- Main subsidy-emission target: **5,259,600 subsidy-bearing blocks**, nominally 10 years at the target interval.
- After block 5,259,600, block subsidy is permanently **0**.
- Transaction fee remains **0.10%** of transferred amount: **0.09% miner share + 0.01% development-treasury share**.
- Treasury remains a normal spendable wallet; ordinary transfers to/from it are valid.
- Block subsidy is paid to the successful block miner's reward address, not to treasury.
- Existing public testnet state and its current 50-SUDH/halving behavior are not modified by this design. Mainnet parameters must be isolated from testnet parameters.

The hard cap is a consensus invariant: cumulative subsidy issuance MUST never exceed 5,100,000,000,000,000 base units.

## 2. Emission shape

Annual issuance targets are:

| Year | % of cap | SUDH issued | Cumulative SUDH |
|---:|---:|---:|---:|
| 1 | 16% | 8,160,000 | 8,160,000 |
| 2 | 14% | 7,140,000 | 15,300,000 |
| 3 | 13% | 6,630,000 | 21,930,000 |
| 4 | 12% | 6,120,000 | 28,050,000 |
| 5 | 11% | 5,610,000 | 33,660,000 |
| 6 | 10% | 5,100,000 | 38,760,000 |
| 7 | 8% | 4,080,000 | 42,840,000 |
| 8 | 7% | 3,570,000 | 46,410,000 |
| 9 | 5% | 2,550,000 | 48,960,000 |
| 10 | 4% | 2,040,000 | 51,000,000 |

Each nominal year is 525,960 blocks. Each quarter is exactly 131,490 blocks. There are 40 subsidy epochs.

Within each year, the four quarterly issuance targets use multipliers 1.03, 1.01, 0.99 and 0.97 against one quarter of that year's annual target. These multipliers sum to 4.00, so each annual target remains exact while subsidy declines within the year.

## 3. Exact 40-epoch schedule

`Base reward` below is the floor reward paid on every block in the epoch. To eliminate division remainder deterministically, the first `Remainder blocks` of that epoch receive exactly **one additional base unit (0.00000001 SUDH)**. This makes every epoch target and the final 51M cap exact without floating-point arithmetic.

| Epoch | Block heights | Epoch issuance SUDH | Base reward SUDH/block | Remainder blocks (+1 base unit) |
|---|---:|---:|---:|---:|
| 1Q1 | 1–131,490 | 2,101,200 | 15.97992242 | 99,420 |
| 1Q2 | 131,491–262,980 | 2,060,400 | 15.66963267 | 22,170 |
| 1Q3 | 262,981–394,470 | 2,019,600 | 15.35934291 | 76,410 |
| 1Q4 | 394,471–525,960 | 1,978,800 | 15.04905315 | 130,650 |
| 2Q1 | 525,961–657,450 | 1,838,550 | 13.98243212 | 54,120 |
| 2Q2 | 657,451–788,940 | 1,802,850 | 13.71092858 | 101,580 |
| 2Q3 | 788,941–920,430 | 1,767,150 | 13.43942505 | 17,550 |
| 2Q4 | 920,431–1,051,920 | 1,731,450 | 13.16792151 | 65,010 |
| 3Q1 | 1,051,921–1,183,410 | 1,707,225 | 12.98368697 | 31,470 |
| 3Q2 | 1,183,411–1,314,900 | 1,674,075 | 12.73157654 | 75,540 |
| 3Q3 | 1,314,901–1,446,390 | 1,640,925 | 12.47946611 | 119,610 |
| 3Q4 | 1,446,391–1,577,880 | 1,607,775 | 12.22735569 | 32,190 |
| 4Q1 | 1,577,881–1,709,370 | 1,575,900 | 11.98494182 | 8,820 |
| 4Q2 | 1,709,371–1,840,860 | 1,545,300 | 11.75222450 | 49,500 |
| 4Q3 | 1,840,861–1,972,350 | 1,514,700 | 11.51950718 | 90,180 |
| 4Q4 | 1,972,351–2,103,840 | 1,484,100 | 11.28678986 | 130,860 |
| 5Q1 | 2,103,841–2,235,330 | 1,444,575 | 10.98619666 | 117,660 |
| 5Q2 | 2,235,331–2,366,820 | 1,416,525 | 10.77287246 | 23,460 |
| 5Q3 | 2,366,821–2,498,310 | 1,388,475 | 10.55954825 | 60,750 |
| 5Q4 | 2,498,311–2,629,800 | 1,360,425 | 10.34622404 | 98,040 |
| 6Q1 | 2,629,801–2,761,290 | 1,313,250 | 9.98745151 | 95,010 |
| 6Q2 | 2,761,291–2,892,780 | 1,287,750 | 9.79352041 | 128,910 |
| 6Q3 | 2,892,781–3,024,270 | 1,262,250 | 9.59958932 | 31,320 |
| 6Q4 | 3,024,271–3,155,760 | 1,236,750 | 9.40565822 | 65,220 |
| 7Q1 | 3,155,761–3,287,250 | 1,050,600 | 7.98996121 | 49,710 |
| 7Q2 | 3,287,251–3,418,740 | 1,030,200 | 7.83481633 | 76,830 |
| 7Q3 | 3,418,741–3,550,230 | 1,009,800 | 7.67967145 | 103,950 |
| 7Q4 | 3,550,231–3,681,720 | 989,400 | 7.52452657 | 131,070 |
| 8Q1 | 3,681,721–3,813,210 | 919,275 | 6.99121606 | 27,060 |
| 8Q2 | 3,813,211–3,944,700 | 901,425 | 6.85546429 | 50,790 |
| 8Q3 | 3,944,701–4,076,190 | 883,575 | 6.71971252 | 74,520 |
| 8Q4 | 4,076,191–4,207,680 | 865,725 | 6.58396075 | 98,250 |
| 9Q1 | 4,207,681–4,339,170 | 656,625 | 4.99372575 | 113,250 |
| 9Q2 | 4,339,171–4,470,660 | 643,875 | 4.89676020 | 130,200 |
| 9Q3 | 4,470,661–4,602,150 | 631,125 | 4.79979466 | 15,660 |
| 9Q4 | 4,602,151–4,733,640 | 618,375 | 4.70282911 | 32,610 |
| 10Q1 | 4,733,641–4,865,130 | 525,300 | 3.99498060 | 90,600 |
| 10Q2 | 4,865,131–4,996,620 | 515,100 | 3.91740816 | 104,160 |
| 10Q3 | 4,996,621–5,128,110 | 504,900 | 3.83983572 | 117,720 |
| 10Q4 | 5,128,111–5,259,600 | 494,700 | 3.76226328 | 131,280 |

For a block at zero-based offset `i` inside an epoch, subsidy is `baseReward + 1` base unit when `i < remainderBlocks`; otherwise it is `baseReward`. Heights greater than 5,259,600 return zero subsidy.

## 4. Consensus and accounting invariants

1. Mainnet genesis creates no spendable premine.
2. Only valid block subsidy can increase circulating SUDH supply; transaction fees redistribute existing SUDH and do not mint supply.
3. For every accepted mainnet block, subsidy must equal the deterministic reward for its height.
4. Cumulative subsidy through height 5,259,600 must equal exactly 51,000,000.00000000 SUDH.
5. Every later block must have zero subsidy.
6. A successful miner receives the block subsidy plus the 0.09% miner fee portions from transactions included in that block.
7. The development treasury receives the 0.01% development fee portion. Ordinary transfers to/from treasury remain valid.
8. Fee rounding must use integer base-unit arithmetic only and preserve existing validated fee semantics unless separately redesigned.
9. No wall-clock timestamp decides reward epochs. Block height is the sole emission selector.

## 5. Network isolation / activation

This is a **mainnet-only** monetary policy. The existing public testnet must remain live and unchanged while implementation is developed and tested on an isolated branch.

Implementation must introduce explicit network-specific monetary parameters or an equivalent consensus-safe selector. A mainnet node must never accidentally use testnet's current 50-SUDH initial subsidy / 1,000,000-block halving schedule, and testnet must never inherit this 51M schedule merely because binaries are upgraded.

Mainnet genesis/network identity and activation are separate deployment decisions. No live node deployment, chain reset, faucet change, treasury movement, or testnet history rewrite is authorized by this design.

## 6. Validation and tests required before mainnet activation

Implementation is not complete until automated tests prove at minimum:

- exact reward at first/last block of every epoch;
- exact +1-base-unit remainder boundary behavior for every epoch;
- exact annual cumulative targets;
- exact final cumulative issuance of 5,100,000,000,000,000 base units;
- reward at height 5,259,601 and all later heights is zero;
- no overflow in supply/reward arithmetic;
- testnet reward behavior remains unchanged;
- miner receives subsidy plus miner-fee share;
- treasury receives only its protocol development-fee credit in fee accounting while ordinary treasury transfers remain valid;
- deterministic results across restart/replay/reorg validation.

## 7. Economic interpretation

The 10-year duration is nominal, not a calendar promise. At a 60-second target, 5,259,600 blocks correspond to 3,155,760,000 seconds, approximately 3652.5 days / 10 target years. If real block production runs faster or slower, wall-clock completion shifts accordingly. Consensus must not compensate by changing issuance based on dates.

This schedule is intended to create declining new-supply pressure and long-lived miner subsidy. It does **not** guarantee market price appreciation. Price depends on demand, utility, liquidity, adoption, security, and market behavior.

After the subsidy ends, network security depends on transaction-fee revenue (currently the 0.09% miner portion). Before mainnet launch, fee-only security at mature expected transaction volumes must be stress-tested and documented; this design does not introduce tail emission.

## 8. Explicit non-goals

- No price guarantee or price-control mechanism.
- No treasury premine or automatic block-subsidy allocation to treasury.
- No burn mechanism in v1.
- No tail emission in v1.
- No modification of the current public testnet chain/history.
- No DEX/CEX liquidity policy in consensus.
