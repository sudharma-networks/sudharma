package demandminer

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestWakeSleeperReturnsEarlyWhenSignaled(t *testing.T) {
	sleeper := NewWakeSleeper()
	done := make(chan error, 1)
	go func() {
		done <- sleeper.Sleep(context.Background(), time.Minute)
	}()

	time.Sleep(20 * time.Millisecond)
	sleeper.Wake()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Sleep: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("sleep was not interrupted by wake signal")
	}
}

func TestWakeServerAcceptsWakeRequests(t *testing.T) {
	sleeper := NewWakeSleeper()
	server, err := NewWakeServer("127.0.0.1:0", sleeper)
	if err != nil {
		t.Fatalf("NewWakeServer: %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		_ = server.Serve(ln)
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	resp, err := http.Post("http://"+ln.Addr().String()+"/v1/wake", "application/json", http.NoBody)
	if err != nil {
		t.Fatalf("POST wake: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body = %s", resp.StatusCode, string(body))
	}

	done := make(chan error, 1)
	go func() {
		done <- sleeper.Sleep(context.Background(), time.Minute)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Sleep: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("wake request did not interrupt supervisor sleep")
	}
}

func TestConfigWakeListenAddressDefaults(t *testing.T) {
	cfg := validConfig()
	if got := cfg.WakeListenAddress(); got != DefaultWakeListen {
		t.Fatalf("WakeListenAddress() = %q", got)
	}
	cfg.WakeListen = "127.0.0.1:29001"
	if got := cfg.WakeListenAddress(); got != "127.0.0.1:29001" {
		t.Fatalf("WakeListenAddress() = %q", got)
	}
}

func TestConfigValidateRejectsPublicWakeListen(t *testing.T) {
	cfg := validConfig()
	cfg.WakeListen = "0.0.0.0:28546"
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "wake_listen") {
		t.Fatalf("expected wake_listen validation error, got %v", err)
	}
}
