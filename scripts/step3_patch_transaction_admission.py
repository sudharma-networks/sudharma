from pathlib import Path


def replace_once(path: str, old: str, new: str) -> None:
    p = Path(path)
    text = p.read_text()
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected exactly one target, found {count}")
    p.write_text(text.replace(old, new, 1))


replace_once(
    "p2p/node.go",
    "\t\tcase MessageTransaction:\n\t\t\ttx, err := DecodeTransaction(message)\n",
    "\t\tcase MessageTransaction:\n\t\t\ttx, err := DecodeTransactionForNetwork(message, n.ActiveNetwork())\n",
)

replace_once(
    "p2p/node.go",
    "\t\t\tstate := n.State()\n\t\t\tif state == nil {\n\t\t\t\tfmt.Printf(\"[TX] Rejected %s: blockchain state unavailable\\n\", tx.ID)\n\t\t\t\tcontinue\n\t\t\t}\n\t\t\tpending := n.mempool.AllTransactions()\n\t\t\tif err := blockchain.ValidateMempoolTransactionFor(\n\t\t\t\tstate,\n\t\t\t\tpending,\n\t\t\t\ttx,\n\t\t\t\tn.ActiveNetwork(),\n\t\t\t); err != nil {\n\t\t\t\tfmt.Printf(\"[TX] Rejected %s from %s: %v\\n\", tx.ID, peer.Info.NodeID, err)\n\t\t\t\tif n.punishPeer(peer, PeerPenaltyInvalidData, \"invalid transaction\") {\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tcontinue\n\t\t\t}\n\t\t\tif err := n.mempool.AddTransaction(tx); err != nil {\n\t\t\t\tfmt.Printf(\"[TX] Mempool add failed for %s: %v\\n\", tx.ID, err)\n\t\t\t\tcontinue\n\t\t\t}\n",
    "\t\t\tif err := n.admitVerifiedTransaction(tx); err != nil {\n\t\t\t\tfmt.Printf(\"[TX] Rejected %s from %s: %v\\n\", tx.ID, peer.Info.NodeID, err)\n\t\t\t\tif n.punishPeer(peer, PeerPenaltyInvalidData, \"invalid transaction\") {\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tcontinue\n\t\t\t}\n",
)

replace_once(
    "p2p/reorg_mempool.go",
    "\t\tpending :=\n\t\t\tn.mempool.AllTransactions()\n",
    "\t\tpending :=\n\t\t\tn.mempool.TransactionsForSender(tx.From)\n",
)

print("Step 3 transaction admission patches applied")
