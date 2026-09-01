Sudharma GPU Miner (Windows)

One click to mine on Sudharma public-testnet.

What you do
1. Double-click "Start Mining.bat"
2. First time only: paste your Sudharma wallet address (40 hex characters)
3. Mining starts automatically and connects to Sudharma public-testnet (seed-1, seed-2 failover via the public RPC proxy)
4. Block rewards go to that wallet address

Next time you open the miner, it remembers your address and starts immediately.

This program never asks for a seed phrase or private key.

Optional
- khushi-miner-nvidia.exe or khushi-miner-opencl.exe in this folder for Khushi GPU hashing
- Without those files the miner still connects and mines public-testnet blocks to your wallet

This is not the demand miner. Both can run at the same time if needed.

Pool mining (optional)
- Double-click "Start Pool Mining.bat" to connect to a Sudharma Stratum pool
- Enter pool URL like stratum+tcp://YOUR_POOL_HOST:3333
- Worker name defaults to rig1 (login becomes wallet.rig1)
- Pool operators run cmd/sudharma-pool; see deployment/testnet/pool.example.json

Mainnet stays closed until launch.
