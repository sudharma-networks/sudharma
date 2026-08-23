package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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

	default:
		fmt.Printf(
			"Unknown command: %s\n\n",
			os.Args[1],
		)
		printUsage()
	}
}

func printUsage() {
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  sudharma-wallet create <wallet-file>")
	fmt.Println("  sudharma-wallet address <wallet-file>")
	fmt.Println("  sudharma-wallet verify <wallet-file>")
	fmt.Println()
}

func createWallet() {
	if len(os.Args) != 3 {
		fmt.Println(
			"Usage: sudharma-wallet create <wallet-file>",
		)
		return
	}

	path := os.Args[2]

	if _, err := os.Stat(path); err == nil {
		fmt.Println(
			"ERROR: wallet file already exists.",
		)
		fmt.Println(
			"Refusing to overwrite existing wallet.",
		)
		return
	}

	password, err :=
		readNewPassword()

	if err != nil {
		fmt.Println(
			"Failed to read password:",
			err,
		)
		return
	}

	w, err :=
		wallet.NewWallet()

	if err != nil {
		fmt.Println(
			"Failed to create wallet:",
			err,
		)
		return
	}

	if err :=
		w.SaveEncrypted(
			path,
			password,
		); err != nil {

		fmt.Println(
			"Failed to save wallet:",
			err,
		)
		return
	}

	fmt.Println()
	fmt.Println("Wallet created successfully.")
	fmt.Println()
	fmt.Println("Address:")
	fmt.Println(w.Address)
	fmt.Println()
	fmt.Println(
		"IMPORTANT: Keep the wallet file and password safely backed up.",
	)
	fmt.Println(
		"Never share the wallet file or password.",
	)
}

func showAddress() {
	if len(os.Args) != 3 {
		fmt.Println(
			"Usage: sudharma-wallet address <wallet-file>",
		)
		return
	}

	path := os.Args[2]

	password, err :=
		readPassword(
			"Wallet password: ",
		)

	if err != nil {
		fmt.Println(
			"Failed to read password:",
			err,
		)
		return
	}

	w, err :=
		wallet.LoadEncrypted(
			path,
			password,
		)

	if err != nil {
		fmt.Println(
			"Failed to open wallet:",
			err,
		)
		return
	}

	fmt.Println()
	fmt.Println("Sudharma Network wallet address:")
	fmt.Println(w.Address)
}

func verifyWallet() {
	if len(os.Args) != 3 {
		fmt.Println(
			"Usage: sudharma-wallet verify <wallet-file>",
		)
		return
	}

	path := os.Args[2]

	password, err :=
		readPassword(
			"Wallet password: ",
		)

	if err != nil {
		fmt.Println(
			"Failed to read password:",
			err,
		)
		return
	}

	w, err :=
		wallet.LoadEncrypted(
			path,
			password,
		)

	if err != nil {
		fmt.Println()
		fmt.Println("Wallet verification FAILED.")
		fmt.Println(err)
		return
	}

	testMessage :=
		[]byte(
			"Sudharma Network wallet verification",
		)

	signature, err :=
		w.Sign(testMessage)

	if err != nil {
		fmt.Println()
		fmt.Println("Wallet verification FAILED.")
		fmt.Println(err)
		return
	}

	if !w.Verify(
		testMessage,
		signature,
	) {
		fmt.Println()
		fmt.Println(
			"Wallet verification FAILED.",
		)
		return
	}

	fmt.Println()
	fmt.Println("Wallet verification SUCCESSFUL.")
	fmt.Println("Address:")
	fmt.Println(w.Address)
}

func readNewPassword() (
	string,
	error,
) {

	first, err :=
		readPassword(
			"Create wallet password: ",
		)

	if err != nil {
		return "", err
	}

	if len(first) < 12 {
		return "",
			fmt.Errorf(
				"password must contain at least 12 characters",
			)
	}

	second, err :=
		readPassword(
			"Confirm wallet password: ",
		)

	if err != nil {
		return "", err
	}

	if first != second {
		return "",
			fmt.Errorf(
				"passwords do not match",
			)
	}

	return first, nil
}

func readPassword(
	prompt string,
) (string, error) {

	fmt.Print(prompt)

	if term.IsTerminal(
		int(os.Stdin.Fd()),
	) {

		passwordBytes, err :=
			term.ReadPassword(
				int(os.Stdin.Fd()),
			)

		fmt.Println()

		if err != nil {
			return "", err
		}

		return strings.TrimSpace(
			string(passwordBytes),
		), nil
	}

	// Fallback for environments where stdin
	// is not an interactive terminal.
	reader :=
		bufio.NewReader(
			os.Stdin,
		)

	value, err :=
		reader.ReadString('\n')

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(value), nil
}
