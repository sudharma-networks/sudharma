package loopback

import (
	"net"
	"testing"
	"time"
)

func TestListenReturnsEphemeralIPv4Loopback(t *testing.T) {
	listener, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address type = %T, want *net.TCPAddr", listener.Addr())
	}
	if !addr.IP.IsLoopback() {
		t.Fatalf("listener IP = %s, want loopback", addr.IP)
	}
	if addr.IP.To4() == nil {
		t.Fatalf("listener IP = %s, want IPv4", addr.IP)
	}
	if !addr.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("listener IP = %s, want 127.0.0.1", addr.IP)
	}
	if addr.Port == 0 {
		t.Fatal("listener retained ephemeral port zero")
	}
}

func TestListenAcceptsRealLocalTCPConnection(t *testing.T) {
	listener, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	client, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("dial loopback listener: %v", err)
	}
	defer client.Close()

	accepted, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept loopback connection: %v", err)
	}
	defer accepted.Close()

	remote, ok := accepted.RemoteAddr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("accepted remote address type = %T, want *net.TCPAddr", accepted.RemoteAddr())
	}
	if !remote.IP.IsLoopback() {
		t.Fatalf("accepted remote IP = %s, want loopback", remote.IP)
	}
}
