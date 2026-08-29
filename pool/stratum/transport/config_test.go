package transport

import (
	"errors"
	"testing"
	"time"
)

func TestNormalizeConfigDefaults(t *testing.T) {
	got, err := normalizeConfig(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if got.readTimeout != 30*time.Second || got.writeTimeout != 10*time.Second || got.refreshInterval != 5*time.Second {
		t.Fatalf("unexpected time defaults: %+v", got)
	}
	if got.maxProtocolErrors != 8 || got.requestsPerSecond != 20 || got.burst != 40 {
		t.Fatalf("unexpected abuse defaults: %+v", got)
	}
}

func TestNormalizeConfigExplicitValues(t *testing.T) {
	cfg := Config{
		ReadTimeout:       time.Second,
		WriteTimeout:      2 * time.Second,
		RefreshInterval:   3 * time.Second,
		MaxProtocolErrors: 4,
		RequestsPerSecond: 5,
		Burst:             6,
	}
	got, err := normalizeConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.readTimeout != cfg.ReadTimeout || got.writeTimeout != cfg.WriteTimeout || got.refreshInterval != cfg.RefreshInterval {
		t.Fatalf("explicit time values changed: %+v", got)
	}
	if got.maxProtocolErrors != cfg.MaxProtocolErrors || got.requestsPerSecond != cfg.RequestsPerSecond || got.burst != cfg.Burst {
		t.Fatalf("explicit abuse values changed: %+v", got)
	}
}

func TestNormalizeConfigRejectsNegativeDurations(t *testing.T) {
	tests := []Config{
		{ReadTimeout: -time.Nanosecond},
		{WriteTimeout: -time.Nanosecond},
		{RefreshInterval: -time.Nanosecond},
		{MaxProtocolErrors: -1},
	}
	for i, cfg := range tests {
		if _, err := normalizeConfig(cfg); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("case %d: error = %v, want ErrInvalidConfig", i, err)
		}
	}
}
