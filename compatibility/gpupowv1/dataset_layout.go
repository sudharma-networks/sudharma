package gpupowv1

import "errors"

type DatasetLocation struct {
	Chunk  uint32
	Offset uint64
}

func (p ProductionMemoryPolicy) DatasetItemLocation(index uint64) (DatasetLocation, error) {
	if err := p.Validate(); err != nil {
		return DatasetLocation{}, err
	}
	itemCount := p.DatasetBytes / p.ItemBytes
	if index >= itemCount {
		return DatasetLocation{}, errors.New("GPU-PoW dataset item index out of range")
	}
	max := ^uint64(0)
	if index > max/p.ItemBytes {
		return DatasetLocation{}, errors.New("GPU-PoW dataset item offset overflow")
	}
	byteOffset := index * p.ItemBytes
	chunk := byteOffset / p.ChunkBytes
	if chunk > uint64(^uint32(0)) {
		return DatasetLocation{}, errors.New("GPU-PoW dataset chunk index overflow")
	}
	return DatasetLocation{
		Chunk:  uint32(chunk),
		Offset: byteOffset % p.ChunkBytes,
	}, nil
}
