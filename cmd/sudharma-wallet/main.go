package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/rpc"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
	"golang.org/x/term"
)

func main() {
	fmt.Println("================================")
	fmt.Println("       Sudharma Network Wallet")
	fmt.Println("================================")

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "create":
		createWallet()
	case "address":
		showAddress()
	case "verify":
		verifyWallet()
	case "node":
		showNodeStatus()
	case "balance":
		showBalance()
	case "send":
		sendTransaction()
	case "tx":
		showTransactionStatus()
	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
	}
}

func printUsage() {
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  sudharma-wallet create <wallet-file>")
	fmt.Println("  sudharma-wallet address <wallet-file>")
	fmt.Println("  sudharma-wallet verify <wallet-file>")
	fmt.Println("  sudharma-wallet node [rpc-url]")
	fmt.Println("  sudharma-wallet balance <wallet-file> [rpc-url]")
	fmt.Println("  sudharma-wallet send <wallet-file> <to-address> <amount-sudh> [rpc-url]")
	fmt.Println("  sudharma-wallet tx <transaction-id> [rpc-url]")
	fmt.Println()
	fmt.Printf("Default RPC: %s\n", rpc.DefaultClientURL)
}

func createWallet() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: sudharma-wallet create <wallet-file>")
		return
	}
	path := os.Args[2]
	if _, err := os.Stat(path); err == nil {
		fmt.Println("ERROR: wallet file already exists.")
		fmt.Println("Refusing to overwrite existing wallet.")
		return
	}
	password, err := readNewPassword()
	if err != nil {
		fmt.Println("Failed to read password:", err)
		return
	}
	w, err := wallet.NewWallet()
	if err != nil {
		fmt.Println("Failed to create wallet:", err)
		return
	}
	if err := w.SaveEncrypted(path, password); err != nil {
		fmt.Println("Failed to save wallet:", err)
		return
	}
	fmt.Println()
	fmt.Println("Wallet created successfully.")
	fmt.Println("Address:")
	fmt.Println(w.Address)
	fmt.Println()
	fmt.Println("IMPORTANT: Keep the wallet file and password safely backed up.")
	fmt.Println("Never share the wallet file or password.")
}

func showAddress() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: sudharma-wallet address <wallet-file>")
		return
	}
	w, err := openWallet(os.Args[2])
	if err != nil {
		fmt.Println("Failed to open wallet:", err)
		return
	}
	fmt.Println()
	fmt.Println("Sudharma Network wallet address:")
	fmt.Println(w.Address)
}

func verifyWallet() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: sudharma-wallet verify <wallet-file>")
		return
	}
	w, err := openWallet(os.Args[2])
	if err != nil {
		fmt.Println("Wallet verification FAILED.")
		fmt.Println(err)
		return
	}
	testMessage := []byte("Sudharma Network wallet verification")
	signature, err := w.Sign(testMessage)
	if err != nil || !w.Verify(testMessage, signature) {
		fmt.Println("Wallet verification FAILED.")
		if err != nil {
			fmt.Println(err)
		}
		return
	}
	fmt.Println()
	fmt.Println("Wallet verification SUCCESSFUL.")
	fmt.Println("Address:")
	fmt.Println(w.Address)
}

func showNodeStatus() {
	if len(os.Args) > 3 {
		fmt.Println("Usage: sudharma-wallet node [rpc-url]")
		return
	}
	client, ctx, cancel, err := rpcContext(optionalRPCURL(2))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cancel()
	status, err := client.Status(ctx)
	if err != nil {
		fmt.Println("RPC status failed:", err)
		return
	}
	fmt.Printf("Node ID:  %s\n", status.NodeID)
	fmt.Printf("Height:   %d\n", status.Height)
	fmt.Printf("Tip:      %s\n", status.TipHash)
	fmt.Printf("Peers:    %d\n", status.Peers)
	fmt.Printf("Mempool:  %d\n", status.Mempool)
}

func showBalance() {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		fmt.Println("Usage: sudharma-wallet balance <wallet-file> [rpc-url]")
		return
	}
	w, err := openWallet(os.Args[2])
	if err != nil {
		fmt.Println("Failed to open wallet:", err)
		return
	}
	client, ctx, cancel, err := rpcContext(optionalRPCURL(3))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cancel()
	account, err := client.Account(ctx, w.Address)
	if err != nil {
		fmt.Println("Balance query failed:", err)
		return
	}
	fmt.Println("Address:", account.Address)
	fmt.Println("Balance:", formatCoinAmount(account.Balance), params.CoinSymbol)
	fmt.Println("Confirmed nonce:", account.ConfirmedNonce)
	fmt.Println("Next nonce:", account.NextNonce)
}

func sendTransaction() {
	if len(os.Args) < 5 || len(os.Args) > 6 {
		fmt.Println("Usage: sudharma-wallet send <wallet-file> <to-address> <amount-sudh> [rpc-url]")
		return
	}
	to := strings.TrimSpace(os.Args[3])
	if to == "" || len(to) > 256 {
		fmt.Println("Invalid receiver address.")
		return
	}
	amount, err := parseCoinAmount(os.Args[4])
	if err != nil || amount == 0 {
		fmt.Println("Invalid amount:", err)
		return
	}
	w, err := openWallet(os.Args[2])
	if err != nil {
		fmt.Println("Failed to open wallet:", err)
		return
	}
	if w.Address == to {
		fmt.Println("Refusing to send to the same wallet address.")
		return
	}
	client, ctx, cancel, err := rpcContext(optionalRPCURL(5))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cancel()
	account, err := client.Account(ctx, w.Address)
	if err != nil {
		fmt.Println("Failed to query sender account:", err)
		return
	}
	fee := transactions.CalculateFee(amount)
	if amount > ^uint64(0)-fee {
		fmt.Println("Amount is too large.")
		return
	}
	total := amount + fee
	if account.Balance < total {
		fmt.Printf("Insufficient balance: have %s %s, need %s %s including fee.\n",
			formatCoinAmount(account.Balance), params.CoinSymbol,
			formatCoinAmount(total), params.CoinSymbol)
		return
	}
	tx := transactions.NewTransaction(w.Address, to, amount, account.NextNonce)
	if err := tx.Sign(w); err != nil {
		fmt.Println("Failed to sign transaction:", err)
		return
	}
	result, err := client.SubmitTransaction(ctx, tx)
	if err != nil {
		fmt.Println("Transaction submission failed:", err)
		return
	}
	fmt.Println("Transaction accepted.")
	fmt.Println("Transaction ID:", result.TransactionID)
	fmt.Println("Amount:", formatCoinAmount(amount), params.CoinSymbol)
	fmt.Println("Fee:", formatCoinAmount(fee), params.CoinSymbol)
	fmt.Println("Nonce:", tx.Nonce)
	fmt.Println("Relayed peers:", result.RelayedPeers)
}

func showTransactionStatus() {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		fmt.Println("Usage: sudharma-wallet tx <transaction-id> [rpc-url]")
		return
	}
	client, ctx, cancel, err := rpcContext(optionalRPCURL(3))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cancel()
	status, err := client.Transaction(ctx, strings.TrimSpace(os.Args[2]))
	if err != nil {
		fmt.Println("Transaction query failed:", err)
		return
	}
	fmt.Println("Status:", status.Status)
	if status.Transaction != nil {
		fmt.Println("Transaction ID:", status.Transaction.ID)
		fmt.Println("From:", status.Transaction.From)
		fmt.Println("To:", status.Transaction.To)
		fmt.Println("Amount:", formatCoinAmount(status.Transaction.Amount), params.CoinSymbol)
		fmt.Println("Fee:", formatCoinAmount(status.Transaction.Fee), params.CoinSymbol)
		fmt.Println("Nonce:", status.Transaction.Nonce)
	}
	if status.BlockHeight != nil {
		fmt.Println("Block height:", *status.BlockHeight)
		fmt.Println("Block hash:", status.BlockHash)
		fmt.Println("Confirmations:", status.Confirmations)
	}
}

func openWallet(path string) (*wallet.Wallet, error) {
	password, err := readPassword("Wallet password: ")
	if err != nil {
		return nil, err
	}
	return wallet.LoadEncrypted(path, password)
}

func rpcContext(baseURL string) (*rpc.Client, context.Context, context.CancelFunc, error) {
	client, err := rpc.NewClient(baseURL)
	if err != nil {
		return nil, nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	return client, ctx, cancel, nil
}

func optionalRPCURL(index int) string {
	if len(os.Args) > index {
		return os.Args[index]
	}
	return rpc.DefaultClientURL
}

func parseCoinAmount(value string) (uint64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		return 0, fmt.Errorf("amount must be a positive decimal number")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid decimal amount")
	}
	wholeText := parts[0]
	if wholeText == "" {
		wholeText = "0"
	}
	whole, err := strconv.ParseUint(wholeText, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid whole amount")
	}
	decimals := coinDecimalPlaces()
	fractionText := ""
	if len(parts) == 2 {
		fractionText = parts[1]
	}
	if len(fractionText) > decimals {
		return 0, fmt.Errorf("amount supports at most %d decimal places", decimals)
	}
	for len(fractionText) < decimals {
		fractionText += "0"
	}
	fraction := uint64(0)
	if fractionText != "" {
		fraction, err = strconv.ParseUint(fractionText, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid fractional amount")
		}
	}
	if whole > ^uint64(0)/params.CoinDecimals {
		return 0, fmt.Errorf("amount is too large")
	}
	atomic := whole * params.CoinDecimals
	if atomic > ^uint64(0)-fraction {
		return 0, fmt.Errorf("amount is too large")
	}
	return atomic + fraction, nil
}

func formatCoinAmount(amount uint64) string {
	decimals := coinDecimalPlaces()
	whole := amount / params.CoinDecimals
	fraction := amount % params.CoinDecimals
	if decimals == 0 {
		return strconv.FormatUint(whole, 10)
	}
	return fmt.Sprintf("%d.%0*d", whole, decimals, fraction)
}

func coinDecimalPlaces() int {
	value := params.CoinDecimals
	places := 0
	for value > 1 && value%10 == 0 {
		places++
		value /= 10
	}
	return places
}

func readNewPassword() (string, error) {
	first, err := readPassword("Create wallet password: ")
	if err != nil {
		return "", err
	}
	if len(first) < 12 {
		return "", fmt.Errorf("password must contain at least 12 characters")
	}
	second, err := readPassword("Confirm wallet password: ")
	if err != nil {
		return "", err
	}
	if first != second {
		return "", fmt.Errorf("passwords do not match")
	}
	return first, nil
}

func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(passwordBytes)), nil
	}
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}
