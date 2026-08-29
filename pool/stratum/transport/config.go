package transport

import (
	"errors"
	"time"

	"github.com/sudharma-networks/sudharma/pool/stratum"
)

const (
	defaultReadTimeout       = 30 * time.Second
	defaultWriteTimeout      = 10 * time.Second
	defaultRefreshInterval   = 5 * time.Second
	defaultMaxProtocolErrors = 8
	defaultRequestsPerSecond = uint32(20)
	defaultBurst             = uint32(40)
	maxRequestBytes          = 64 * 1024
)

var (
	ErrInvalidConfig  = errors.New("invalid Stratum transport configuration")
	ErrLineTooLong    = errors.New("Stratum request line too long")
	ErrProtocolBudget = errors.New("Stratum protocol error budget exceeded")
	ErrRateLimited    = errors.New("Stratum connection rate limit exceeded")
)

type SessionFactory func() (*stratum.Session, error)

type Config struct {
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	RefreshInterval   time.Duration
	MaxProtocolErrors int
	RequestsPerSecond uint32
	Burst             uint32
}

type normalizedConfig struct {
	readTimeout       time.Duration
	writeTimeout      time.Duration
	refreshInterval   time.Duration
	maxProtocolErrors int
	requestsPerSecond uint32
	burst             uint32
}

func normalizeConfig(config Config) (normalizedConfig, error) {
	if config.ReadTimeout < 0 || config.WriteTimeout < 0 || config.RefreshInterval < 0 || config.MaxProtocolErrors < 0 {
		return normalizedConfig{}, ErrInvalidConfig
	}

	result := normalizedConfig{
		readTimeout:       config.ReadTimeout,
		writeTimeout:      config.WriteTimeout,
		refreshInterval:   config.RefreshInterval,
		maxProtocolErrors: config.MaxProtocolErrors,
		requestsPerSecond: config.RequestsPerSecond,
		burst:             config.Burst,
	}
	if result.readTimeout == 0 {
		result.readTimeout = defaultReadTimeout
	}
	if result.writeTimeout == 0 {
		result.writeTimeout = defaultWriteTimeout
	}
	if result.refreshInterval == 0 {
		result.refreshInterval = defaultRefreshInterval
	}
	if result.maxProtocolErrors == 0 {
		result.maxProtocolErrors = defaultMaxProtocolErrors
	}
	if result.requestsPerSecond == 0 {
		result.requestsPerSecond = defaultRequestsPerSecond
	}
	if result.burst == 0 {
		result.burst = defaultBurst
	}
	if result.requestsPerSecond == 0 || result.burst == 0 {
		return normalizedConfig{}, ErrInvalidConfig
	}
	return result, nil
}
