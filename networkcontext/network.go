package networkcontext

import (
	"sync"

	"github.com/sudharma-networks/sudharma/params"
)

var active = struct {
	sync.RWMutex
	network params.NetworkID
}{network: params.DefaultNetwork}

// Set binds the process-wide active network used by convenience transaction
// signing paths. Sudharma runs one chain namespace per process, so this value is
// selected once during node startup and read by transaction creators.
func Set(network params.NetworkID) {
	active.Lock()
	active.network = network
	active.Unlock()
}

// Active returns the network identity currently bound to this process.
func Active() params.NetworkID {
	active.RLock()
	network := active.network
	active.RUnlock()
	return network
}

// Reset restores the public-testnet default. It is primarily used by tests.
func Reset() {
	Set(params.DefaultNetwork)
}
