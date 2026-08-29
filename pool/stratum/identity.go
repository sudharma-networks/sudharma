package stratum

import (
	"errors"
	"strings"
)

var errInvalidWorkerIdentity = errors.New("invalid worker identity")

func ParseWorkerIdentity(value string) (WorkerIdentity, error) {
	if len(value) == 0 || len(value) > maxIdentityBytes {
		return WorkerIdentity{}, errInvalidWorkerIdentity
	}

	wallet, worker, ok := strings.Cut(value, ".")
	if !ok || strings.Contains(worker, ".") {
		return WorkerIdentity{}, errInvalidWorkerIdentity
	}
	if len(wallet) != 40 || len(worker) == 0 || len(worker) > maxWorkerBytes {
		return WorkerIdentity{}, errInvalidWorkerIdentity
	}

	for i := 0; i < len(wallet); i++ {
		c := wallet[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return WorkerIdentity{}, errInvalidWorkerIdentity
		}
	}
	for i := 0; i < len(worker); i++ {
		c := worker[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return WorkerIdentity{}, errInvalidWorkerIdentity
		}
	}

	return WorkerIdentity{Wallet: wallet, Worker: worker}, nil
}
