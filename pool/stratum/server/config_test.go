package server

import (
	"crypto/tls"
	"errors"
	"testing"
	"time"
)

func TestNormalizeConfigDefaults(t *testing.T) {
	got, err := normalizeConfig(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if got.maxConnections != 256 || got.maxConnectionsPerIP != 8 {
		t.Fatalf("unexpected connection defaults: %+v", got)
	}
	if got.tlsHandshakeTimeout != 10*time.Second || got.acceptErrorBackoff != 100*time.Millisecond {
		t.Fatalf("unexpected time defaults: %+v", got)
	}
	if got.tlsConfig != nil {
		t.Fatal("unexpected TLS config")
	}
}

func TestNormalizeConfigExplicitValues(t *testing.T) {
	cfg := Config{
		MaxConnections:      10,
		MaxConnectionsPerIP: 3,
		TLSHandshakeTimeout: 2 * time.Second,
		AcceptErrorBackoff:  25 * time.Millisecond,
	}
	got, err := normalizeConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.maxConnections != cfg.MaxConnections || got.maxConnectionsPerIP != cfg.MaxConnectionsPerIP {
		t.Fatalf("explicit limits changed: %+v", got)
	}
	if got.tlsHandshakeTimeout != cfg.TLSHandshakeTimeout || got.acceptErrorBackoff != cfg.AcceptErrorBackoff {
		t.Fatalf("explicit durations changed: %+v", got)
	}
}

func TestNormalizeConfigRejectsInvalidLimitsAndDurations(t *testing.T) {
	tests := []Config{
		{MaxConnections: -1},
		{MaxConnectionsPerIP: -1},
		{TLSHandshakeTimeout: -time.Nanosecond},
		{AcceptErrorBackoff: -time.Nanosecond},
		{MaxConnections: 2, MaxConnectionsPerIP: 3},
	}
	for i, cfg := range tests {
		if _, err := normalizeConfig(cfg); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("case %d: error = %v, want ErrInvalidConfig", i, err)
		}
	}
}

func TestNormalizeConfigClonesTLSAndEnforcesMinimum(t *testing.T) {
	original := &tls.Config{}
	got, err := normalizeConfig(Config{TLSConfig: original})
	if err != nil {
		t.Fatal(err)
	}
	if got.tlsConfig == original {
		t.Fatal("TLS config was not cloned")
	}
	if got.tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("minimum TLS version = %x, want TLS 1.2", got.tlsConfig.MinVersion)
	}
	if original.MinVersion != 0 {
		t.Fatalf("caller TLS config mutated: %x", original.MinVersion)
	}
}

func TestNormalizeConfigRejectsTLSBelow12(t *testing.T) {
	_, err := normalizeConfig(Config{TLSConfig: &tls.Config{MinVersion: tls.VersionTLS11}})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("error = %v, want ErrInvalidConfig", err)
	}
}

func TestNormalizeConfigAcceptsTLS12And13(t *testing.T) {
	for _, version := range []uint16{tls.VersionTLS12, tls.VersionTLS13} {
		got, err := normalizeConfig(Config{TLSConfig: &tls.Config{MinVersion: version}})
		if err != nil {
			t.Fatalf("version %x: %v", version, err)
		}
		if got.tlsConfig.MinVersion != version {
			t.Fatalf("version %x normalized to %x", version, got.tlsConfig.MinVersion)
		}
	}
}
