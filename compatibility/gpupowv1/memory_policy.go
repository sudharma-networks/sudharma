package gpupowv1

import "errors"

// ProductionMemoryPolicy describes the pre-activation GPU-PoW v1 memory
// layout. It is kept in the compatibility package until hardware and
// activation gates approve it for consensus use.
type ProductionMemoryPolicy struct {
	DatasetBytes              uint64
	CacheBytes                uint64
	RuntimeReserveBytes       uint64
	ChunkBytes                uint64
	ItemBytes                 uint64
	MinimumDedicatedVRAMBytes uint64
}

var GPUV1ProductionMemory = ProductionMemoryPolicy{
	DatasetBytes:              2 << 30,
	CacheBytes:                16 << 20,
	RuntimeReserveBytes:       256 << 20,
	ChunkBytes:                256 << 20,
	ItemBytes:                 64,
	MinimumDedicatedVRAMBytes: 4 << 30,
}

func (p ProductionMemoryPolicy) RequiredVRAMBytes() (uint64, bool) {
	max := ^uint64(0)
	if p.DatasetBytes > max-p.CacheBytes {
		return 0, false
	}
	required := p.DatasetBytes + p.CacheBytes
	if required > max-p.RuntimeReserveBytes {
		return 0, false
	}
	return required + p.RuntimeReserveBytes, true
}

func (p ProductionMemoryPolicy) EligibleDedicatedVRAM(total, available uint64) bool {
	required, ok := p.RequiredVRAMBytes()
	return ok && total >= p.MinimumDedicatedVRAMBytes && available >= required
}

func (p ProductionMemoryPolicy) Validate() error {
	if p.DatasetBytes == 0 || p.CacheBytes == 0 || p.RuntimeReserveBytes == 0 ||
		p.ChunkBytes == 0 || p.ItemBytes == 0 || p.MinimumDedicatedVRAMBytes == 0 {
		return errors.New("GPU-PoW memory policy values must be positive")
	}
	required, ok := p.RequiredVRAMBytes()
	if !ok {
		return errors.New("GPU-PoW memory policy size overflow")
	}
	if p.DatasetBytes%p.ItemBytes != 0 {
		return errors.New("GPU-PoW dataset size must be divisible by item size")
	}
	if p.ChunkBytes%p.ItemBytes != 0 {
		return errors.New("GPU-PoW chunk size must be divisible by item size")
	}
	if p.DatasetBytes%p.ChunkBytes != 0 {
		return errors.New("GPU-PoW dataset size must be divisible by chunk size")
	}
	if required > p.MinimumDedicatedVRAMBytes {
		return errors.New("GPU-PoW required memory exceeds minimum VRAM target")
	}
	return nil
}
