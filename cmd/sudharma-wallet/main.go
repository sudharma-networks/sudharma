package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/sudharma-networks/sudharma/rpc"
	"golang.org/x/term"
)

var stdinPasswordReader struct {
	mu     sync.Mutex
	file   *os.File
	reader *bufio.Reader
}

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
	case "backup":
		backupWallet()
	case "passwd":
		changeWalletPassword()
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
	fmt.Println("  sudharma-wallet backup <wallet-file> <backup-file>")
	fmt.Println("  sudharma-wallet passwd <wallet-file>")
	fmt.Println("  sudharma-wallet node [rpc-url]")
	fmt.Println("  sudharma-wallet balance <wallet-file> [rpc-url]")
	fmt.Println("  sudharma-wallet send <wallet-file> <to-address> <amount-sudh> [rpc-url]")
	fmt.Println("  sudharma-wallet tx <transaction-id> [rpc-url]")
	fmt.Println()
	fmt.Printf("Default RPC: %s\n", rpc.DefaultClientURL)
}

func readNewPassword() (string, error) {
	return readNewPasswordWithPrompts("Create wallet password: ", "Confirm wallet password: ")
}

func readNewPasswordWithPrompts(firstPrompt, secondPrompt string) (string, error) {
	first, err := readPassword(firstPrompt)
	if err != nil {
		return "", err
	}
	if len(first) < 12 {
		return "", fmt.Errorf("password must contain at least 12 characters")
	}
	second, err := readPassword(secondPrompt)
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

	stdinPasswordReader.mu.Lock()
	defer stdinPasswordReader.mu.Unlock()

	if stdinPasswordReader.reader == nil || stdinPasswordReader.file != os.Stdin {
		stdinPasswordReader.file = os.Stdin
		stdinPasswordReader.reader = bufio.NewReader(os.Stdin)
	}

	value, err := stdinPasswordReader.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}
