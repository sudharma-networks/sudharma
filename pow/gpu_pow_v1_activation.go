package pow

import "github.com/sudharma-networks/sudharma/blockchain"

// GPUV1VersionAllowedAtHeight applies the Version-1/Version-2 activation
// boundary without selecting a network or arming an activation height.
// Before activation only legacy Version 1 is valid; from activation onward
// only GPU-PoW Version 2 is valid. Future versions are rejected explicitly.
func GPUV1VersionAllowedAtHeight(version uint32, height, activationHeight uint64) bool {
	return blockchain.PoWPolicy{
		GPUV1ActivationHeight: activationHeight,
	}.VersionAllowed(version, height)
}
