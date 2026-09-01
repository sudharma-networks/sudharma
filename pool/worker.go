package pool

import (
	"fmt"
	"regexp"
	"strings"
)

var walletPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// WorkerIdentity is parsed from Stratum login strings such as wallet.worker.
type WorkerIdentity struct {
	Address    string
	WorkerName string
	Login      string
}

// ParseWorkerIdentity accepts wallet or wallet.worker forms used by solo and pool miners.
func ParseWorkerIdentity(login string) (WorkerIdentity, error) {
	raw := strings.TrimSpace(login)
	if raw == "" {
		return WorkerIdentity{}, fmt.Errorf("worker login required")
	}

	parts := strings.SplitN(raw, ".", 2)
	address := strings.ToLower(strings.TrimSpace(parts[0]))
	if !walletPattern.MatchString(address) {
		return WorkerIdentity{}, fmt.Errorf("worker login must start with a 40-character wallet address")
	}

	name := "default"
	if len(parts) == 2 {
		name = strings.TrimSpace(parts[1])
		if name == "" {
			return WorkerIdentity{}, fmt.Errorf("worker name cannot be empty")
		}
	}

	return WorkerIdentity{
		Address:    address,
		WorkerName: name,
		Login:      raw,
	}, nil
}
