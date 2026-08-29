package server

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/pool/stratum/transport"
)

type preparedConnResult struct {
	conn net.Conn
	err  error
}

func TestPrepareConnTLSHandshake(t *testing.T) {
	serverRaw, clientRaw := net.Pipe()
	defer clientRaw.Close()

	normalized, err := normalizeConfig(Config{
		TLSConfig:           newTestServerTLSConfig(t),
		TLSHandshakeTimeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan preparedConnResult, 1)
	go func() {
		conn, err := prepareConn(context.Background(), serverRaw, normalized)
		done <- preparedConnResult{conn: conn, err: err}
	}()

	client := tls.Client(clientRaw, newTestClientTLSConfig())
	if err := client.Handshake(); err != nil {
		t.Fatalf("client TLS handshake: %v", err)
	}
	select {
	case result := <-done:
		if result.err != nil {
			t.Fatalf("prepareConn error = %v", result.err)
		}
		if _, ok := result.conn.(*tls.Conn); !ok {
			t.Fatalf("prepared connection type = %T, want *tls.Conn", result.conn)
		}
		_ = result.conn.Close()
	case <-time.After(time.Second):
		t.Fatal("prepareConn did not finish TLS handshake")
	}
}

func TestPrepareConnRejectsPlaintextTLSHandshake(t *testing.T) {
	serverRaw, clientRaw := net.Pipe()
	defer clientRaw.Close()

	normalized, err := normalizeConfig(Config{
		TLSConfig:           newTestServerTLSConfig(t),
		TLSHandshakeTimeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan preparedConnResult, 1)
	go func() {
		conn, err := prepareConn(context.Background(), serverRaw, normalized)
		done <- preparedConnResult{conn: conn, err: err}
	}()
	if _, err := io.WriteString(clientRaw, "not tls\n"); err != nil {
		t.Fatal(err)
	}

	select {
	case result := <-done:
		if result.conn != nil {
			t.Fatalf("prepared connection = %T, want nil", result.conn)
		}
		if result.err == nil || !strings.Contains(result.err.Error(), "handshake Stratum TLS") {
			t.Fatalf("prepareConn error = %v, want TLS handshake context", result.err)
		}
	case <-time.After(time.Second):
		t.Fatal("prepareConn did not reject plaintext client")
	}
}

func TestPrepareConnTLSHandshakeTimeout(t *testing.T) {
	serverRaw, clientRaw := net.Pipe()
	defer clientRaw.Close()

	normalized, err := normalizeConfig(Config{
		TLSConfig:           newTestServerTLSConfig(t),
		TLSHandshakeTimeout: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan preparedConnResult, 1)
	go func() {
		conn, err := prepareConn(context.Background(), serverRaw, normalized)
		done <- preparedConnResult{conn: conn, err: err}
	}()

	select {
	case result := <-done:
		if result.conn != nil {
			t.Fatalf("prepared connection = %T, want nil", result.conn)
		}
		if result.err == nil || !strings.Contains(result.err.Error(), "handshake Stratum TLS") {
			t.Fatalf("prepareConn error = %v, want TLS handshake context", result.err)
		}
	case <-time.After(time.Second):
		t.Fatal("TLS handshake timeout did not terminate prepareConn")
	}
}

func TestServeListenerTLSDelegatesToStageE(t *testing.T) {
	serverRaw, clientRaw := net.Pipe()
	listener := newScriptedListener(1)
	listener.push(&addressedConn{
		Conn:   serverRaw,
		remote: &net.TCPAddr{IP: net.ParseIP("203.0.113.20"), Port: 5000},
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls atomic.Int32
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- ServeListener(
			ctx,
			listener,
			newServerFactory(&calls),
			transport.Config{},
			Config{TLSConfig: newTestServerTLSConfig(t), TLSHandshakeTimeout: time.Second},
		)
	}()

	client := tls.Client(clientRaw, newTestClientTLSConfig())
	defer client.Close()
	if err := client.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := client.Handshake(); err != nil {
		t.Fatalf("client TLS handshake: %v", err)
	}
	if err := client.SetDeadline(time.Time{}); err != nil {
		t.Fatal(err)
	}

	if _, err := io.WriteString(client, `{"id":1,"method":"mining.subscribe","params":[]}`+"\n"); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(client)
	if id := int(readServerJSONLine(t, reader)["id"].(float64)); id != 1 {
		t.Fatalf("subscribe response id = %d, want 1", id)
	}
	authorize := `{"id":2,"method":"mining.authorize","params":["` + serverWallet + `.rig_tls","x"]}` + "\n"
	if _, err := io.WriteString(client, authorize); err != nil {
		t.Fatal(err)
	}
	if got := readServerJSONLine(t, reader)["result"]; got != true {
		t.Fatalf("authorize result = %v, want true", got)
	}
	if got := readServerJSONLine(t, reader)["method"]; got != "mining.set_difficulty" {
		t.Fatalf("first work method = %v", got)
	}
	if got := readServerJSONLine(t, reader)["method"]; got != "mining.notify" {
		t.Fatalf("second work method = %v", got)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("session factory calls = %d, want 1", got)
	}

	cancel()
	select {
	case err := <-serveDone:
		if err != context.Canceled {
			t.Fatalf("ServeListener error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeListener did not stop after TLS test cancellation")
	}
}

func newTestServerTLSConfig(t *testing.T) *tls.Config {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
	}
}

func newTestClientTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	}
}
