package p2p

import (
	"bufio"
	"errors"
	"fmt"
)

const (
	// MaxPeerMessageBytes bounds one newline-delimited P2P frame. This is a
	// transport safety limit; peers that need larger messages must use a future
	// chunked protocol rather than forcing unbounded buffering.
	MaxPeerMessageBytes = 16 << 20 // 16 MiB
)

var ErrPeerMessageTooLarge = errors.New("peer message exceeds maximum size")

// readBoundedPeerMessage reads one newline-delimited frame without allowing
// the backing buffer to grow beyond MaxPeerMessageBytes.
func readBoundedPeerMessage(reader *bufio.Reader) ([]byte, error) {
	if reader == nil {
		return nil, fmt.Errorf("peer reader cannot be nil")
	}

	message := make([]byte, 0, 4096)
	for {
		fragment, err := reader.ReadSlice('\n')
		if len(message)+len(fragment) > MaxPeerMessageBytes {
			return nil, ErrPeerMessageTooLarge
		}
		message = append(message, fragment...)

		switch {
		case err == nil:
			return message, nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		default:
			return nil, err
		}
	}
}
