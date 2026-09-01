from pathlib import Path

path = Path("cmd/sudharmad/main.go")
text = path.read_text(encoding="utf-8")

replacements = [
    (
        "\tloadedChain, err :=\n\t\tblockchain.LoadChainFromFile(\n\t\t\tchainPath,\n\t\t)\n",
        "\tloadedChain, err :=\n\t\tloadChainForNetwork(\n\t\t\tchainPath,\n\t\t\tnetwork,\n\t\t)\n",
    ),
    (
        "\t\tif chain.Height() == 0 {\n\t\t\tnodeState =\n\t\t\t\tblockchain.NewState()\n",
        "\t\tif chain.Height() == 0 {\n\t\t\tnodeState =\n\t\t\t\tnewGenesisStateForPolicy(monetaryPolicy)\n",
    ),
    (
        "\tstate :=\n\t\tblockchain.NewState()\n\n\tfor height := uint64(1); height <= chain.Height(); height++ {\n",
        "\tstate :=\n\t\tnewGenesisStateForPolicy(policy)\n\n\tfor height := uint64(1); height <= chain.Height(); height++ {\n",
    ),
]

for old, new in replacements:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"expected exactly one match, got {count}: {old!r}")
    text = text.replace(old, new, 1)

path.write_text(text, encoding="utf-8")
