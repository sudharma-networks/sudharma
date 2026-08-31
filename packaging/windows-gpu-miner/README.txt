Sudharma GPU Miner (Windows)

This is the public GPU miner. It is a separate program from the demand miner.
The demand miner is not changed by this app. Both can run at the same time.

Sudharma is GPU-mined only. There is no CPU miner product and no ASIC miner.
This is true for public-testnet and for mainnet.

What you need
- A 64-bit Windows PC
- Your Sudharma wallet address (40 lowercase hex characters)
- Optional: an NVIDIA CUDA or AMD/OpenCL GPU hasher in this folder
  khushi-miner-nvidia.exe  (NVIDIA)
  or khushi-miner-opencl.exe (AMD / OpenCL)

Start mining
1. Double-click "Start Mining.bat"
2. Paste your wallet address
3. Mining starts on public-testnet
4. Block rewards go to that wallet address

This program never asks for a seed phrase or private key.
It never starts the demand miner.

If the Khushi GPU hasher is not in this folder, the miner still submits
public-testnet blocks to your pasted address. It will not switch to a
CPU or ASIC mining product.

Mainnet stays closed until launch. When it opens, it will still be GPU-only.

Do not use CPU miners, ASIC firmware, or unofficial "easy hash" tools.
