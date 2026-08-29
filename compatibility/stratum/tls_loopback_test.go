package stratumcompat

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/pool/stratum"
	"github.com/sudharma-networks/sudharma/pool/stratum/loopback"
	"github.com/sudharma-networks/sudharma/pool/stratum/server"
)

func TestLoopbackTLSRejectsPlaintextThenServesTranscript(t *testing.T) {
	listener, err := loopback.Listen()
	if err != nil {
		t.Fatal(err)
	}

	serverTLS, clientTLS := newCompatTLSConfigs(t)
	source := newCompatSource()
	var factoryCalls atomic.Int32
	cancel, done := startCompatServer(
		t,
		listener,
		newCompatFactory(t, source, &factoryCalls),
		server.Config{TLSConfig: serverTLS, TLSHandshakeTimeout: time.Second},
	)

	plaintext := dialCompatTCP(t, listener.Addr().String())
	_, _ = fmt.Fprintln(plaintext, `{"id":1,"method":"mining.subscribe","params":[]}`)
	var one [1]byte
	_ = plaintext.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := plaintext.Read(one[:]); err == nil {
		_ = plaintext.Close()
		t.Fatal("plaintext client remained open on TLS-enabled loopback listener")
	}
	_ = plaintext.Close()
	if got := factoryCalls.Load(); got != 0 {
		t.Fatalf("session factory calls after plaintext TLS failure = %d, want 0", got)
	}

	dialer := &net.Dialer{Timeout: time.Second}
	client, err := tls.DialWithDialer(dialer, "tcp4", listener.Addr().String(), clientTLS)
	if err != nil {
		t.Fatalf("TLS dial loopback Stratum: %v", err)
	}
	if err := client.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		_ = client.Close()
		t.Fatal(err)
	}
	if state := client.ConnectionState(); state.Version < tls.VersionTLS12 {
		_ = client.Close()
		t.Fatalf("TLS version = %#x, want TLS 1.2 or newer", state.Version)
	}
	reader := bufio.NewReader(client)

	writeRequest(t, client, `{"id":1,"method":"mining.subscribe","params":["khushi-tls-loopback/1.0"]}`)
	subscribe := readJSONMessage(t, reader)
	if subscribe["id"] != float64(1) || subscribe["error"] != nil {
		_ = client.Close()
		t.Fatalf("subscribe response = %#v", subscribe)
	}

	worker := compatWallet + ".rig_tls"
	writeRequest(t, client, fmt.Sprintf(`{"id":2,"method":"mining.authorize","params":[%q,"x"]}`, worker))
	requireResponseResult(t, reader, 2, true)
	jobID, lane := requireWorkMessages(t, reader)

	shareNonce := uint64(lane)<<32 | 1
	blockNonce := uint64(lane)<<32 | 2
	writeRequest(t, client, fmt.Sprintf(`{"id":3,"method":"mining.submit","params":[%q,%q,%q]}`, worker, jobID, fmt.Sprintf("%016x", shareNonce)))
	requireResponseResult(t, reader, 3, string(stratum.SubmitAcceptedShare))
	writeRequest(t, client, fmt.Sprintf(`{"id":4,"method":"mining.submit","params":[%q,%q,%q]}`, worker, jobID, fmt.Sprintf("%016x", blockNonce)))
	requireResponseResult(t, reader, 4, string(stratum.SubmitAcceptedBlock))

	if got := factoryCalls.Load(); got != 1 {
		_ = client.Close()
		t.Fatalf("session factory calls = %d, want 1", got)
	}
	submissions := source.submitted()
	if len(submissions) != 1 || submissions[0].Nonce != blockNonce {
		_ = client.Close()
		t.Fatalf("TLS network candidate submissions = %#v, want one block nonce %016x", submissions, blockNonce)
	}

	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	stopCompatServer(t, cancel, done)
}

func newCompatTLSConfigs(t *testing.T) (*tls.Config, *tls.Config) {
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
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(certificate)
	serverConfig := &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
	}
	clientConfig := &tls.Config{
		RootCAs:    roots,
		ServerName: "localhost",
		MinVersion: tls.VersionTLS12,
	}
	return serverConfig, clientConfig
}
