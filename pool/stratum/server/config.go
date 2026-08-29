package server

import (
	"crypto/tls"
	"errors"
	"time"
)

const (
	defaultMaxConnections      = 256
	defaultMaxConnectionsPerIP = 8
	defaultTLSHandshakeTimeout = 10 * time.Second
	defaultAcceptErrorBackoff  = 100 * time.Millisecond
)

var ErrInvalidConfig = errors.New("invalid Stratum server configuration")

type Config struct {
	MaxConnections      int
	MaxConnectionsPerIP int
	TLSConfig           *tls.Config
	TLSHandshakeTimeout time.Duration
	AcceptErrorBackoff  time.Duration
}

type normalizedConfig struct {
	maxConnections      int
	maxConnectionsPerIP int
	tlsConfig           *tls.Config
	tlsHandshakeTimeout time.Duration
	acceptErrorBackoff  time.Duration
}

func normalizeConfig(config Config) (normalizedConfig, error) {
	if config.MaxConnections < 0 || config.MaxConnectionsPerIP < 0 || config.TLSHandshakeTimeout < 0 || config.AcceptErrorBackoff < 0 {
		return normalizedConfig{}, ErrInvalidConfig
	}

	result := normalizedConfig{
		maxConnections:      config.MaxConnections,
		maxConnectionsPerIP: config.MaxConnectionsPerIP,
		tlsHandshakeTimeout: config.TLSHandshakeTimeout,
		acceptErrorBackoff:  config.AcceptErrorBackoff,
	}
	if result.maxConnections == 0 {
		result.maxConnections = defaultMaxConnections
	}
	if result.maxConnectionsPerIP == 0 {
		result.maxConnectionsPerIP = defaultMaxConnectionsPerIP
	}
	if result.tlsHandshakeTimeout == 0 {
		result.tlsHandshakeTimeout = defaultTLSHandshakeTimeout
	}
	if result.acceptErrorBackoff == 0 {
		result.acceptErrorBackoff = defaultAcceptErrorBackoff
	}
	if result.maxConnectionsPerIP > result.maxConnections {
		return normalizedConfig{}, ErrInvalidConfig
	}

	if config.TLSConfig != nil {
		result.tlsConfig = config.TLSConfig.Clone()
		if result.tlsConfig.MinVersion == 0 {
			result.tlsConfig.MinVersion = tls.VersionTLS12
		}
		if result.tlsConfig.MinVersion < tls.VersionTLS12 {
			return normalizedConfig{}, ErrInvalidConfig
		}
	}

	return result, nil
}
