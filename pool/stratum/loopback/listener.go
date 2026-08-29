package loopback

import (
	"errors"
	"fmt"
	"net"
)

const (
	loopbackNetwork = "tcp4"
	loopbackAddress = "127.0.0.1:0"
)

var ErrUnsafeAddress = errors.New("unsafe Stratum loopback listener address")

// Listen opens an IPv4 loopback-only listener on an operating-system-selected
// ephemeral port. The address is intentionally not configurable.
func Listen() (net.Listener, error) {
	listener, err := net.Listen(loopbackNetwork, loopbackAddress)
	if err != nil {
		return nil, fmt.Errorf("listen on Stratum loopback: %w", err)
	}

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok || addr.IP == nil || !addr.IP.IsLoopback() || addr.IP.To4() == nil || addr.Port == 0 {
		_ = listener.Close()
		return nil, ErrUnsafeAddress
	}
	if !addr.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		_ = listener.Close()
		return nil, ErrUnsafeAddress
	}
	return listener, nil
}
