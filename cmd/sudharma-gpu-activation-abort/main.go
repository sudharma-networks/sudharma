package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sudharma-networks/sudharma/operations"
)

func main() {
	if err := runAbortCLI(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "sudharma-gpu-activation-abort: %v\n", err)
		os.Exit(1)
	}
}

func runAbortCLI(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("sudharma-gpu-activation-abort", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dataDirectory := flags.String("data-dir", "", "stopped node data directory")
	evidenceDirectory := flags.String("evidence-dir", "", "new evidence directory")
	expectedHeight := flags.Uint64("expected-activation-height", 0, "exact persisted activation height")
	confirm := flags.Bool("confirm-abort", false, "confirm the stopped-node pre-boundary abort")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *dataDirectory == "" {
		return fmt.Errorf("data-dir is required")
	}
	if *evidenceDirectory == "" {
		return fmt.Errorf("evidence-dir is required")
	}
	heightProvided := false
	flags.Visit(func(current *flag.Flag) {
		if current.Name == "expected-activation-height" {
			heightProvided = true
		}
	})
	if !heightProvided {
		return fmt.Errorf("expected-activation-height is required")
	}
	if !*confirm {
		return fmt.Errorf("confirm-abort is required")
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}

	evidence, err := operations.AbortGPUActivation(operations.GPUActivationAbortOptions{
		DataDirectory:            *dataDirectory,
		EvidenceDirectory:        *evidenceDirectory,
		ExpectedActivationHeight: *expectedHeight,
	})
	if err != nil {
		return err
	}
	fmt.Fprintln(output, "activation_abort=completed")
	fmt.Fprintf(output, "chain_tip_height=%d\n", evidence.ChainTipHeight)
	fmt.Fprintf(output, "activation_height=%d\n", evidence.ActivationHeight)
	fmt.Fprintf(output, "evidence_directory=%s\n", *evidenceDirectory)
	fmt.Fprintln(output, "consensus_activation=disabled-after-config-update-and-restart")
	return nil
}
