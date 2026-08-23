package p2p

import (
	"fmt"
	"sync"
	"time"
)

var partitionRecoveryLocks sync.Map

func partitionRecoveryLock(n *Node) *sync.Mutex {
	if lock, ok := partitionRecoveryLocks.Load(n); ok {
		return lock.(*sync.Mutex)
	}
	lock := &sync.Mutex{}
	actual, _ := partitionRecoveryLocks.LoadOrStore(n, lock)
	return actual.(*sync.Mutex)
}

// RecoverNetworkOnce performs one bounded live recovery cycle. It first retries
// previously discovered addresses, then asks the best eligible connected peers
// to reconcile the chain through the existing fully validating sync path.
// Only one recovery cycle may run per node at a time.
func (n *Node) RecoverNetworkOnce(timeout time.Duration) (*PartitionRecoveryResult, error) {
	if n == nil {
		return nil, fmt.Errorf("node cannot be nil")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("recovery timeout must be greater than zero")
	}

	lock := partitionRecoveryLock(n)
	if !lock.TryLock() {
		return nil, fmt.Errorf("network recovery is already in progress")
	}
	defer lock.Unlock()

	// A recovered discovered connection can immediately become a candidate for
	// chain reconciliation. Failures are intentionally non-fatal because other
	// connected peers may still provide a valid recovery path.
	n.AutoConnectDiscoveredPeers()

	if n.PeerCount() == 0 {
		return &PartitionRecoveryResult{Failures: make(map[string]string)},
			fmt.Errorf("no connected peers available after recovery attempts")
	}

	return n.SyncFromBestAvailablePeer(timeout)
}

func clearPartitionRecoveryState(n *Node) {
	if n != nil {
		partitionRecoveryLocks.Delete(n)
	}
}
