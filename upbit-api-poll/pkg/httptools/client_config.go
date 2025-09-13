package httptools

import (
	"log/slog"
	"time"
)

type ClientConfig struct {
	Proxy   *Proxy
	Metrics Metrics
	Logger  *slog.Logger
	ClientRetriesConfig
}

type ClientRetriesConfig struct {
	MaxRetries           int           `mapstructure:"max_retries"`
	RetryDelay           time.Duration `mapstructure:"retry_delay"`
	RetryDelayMultiplier float64       `mapstructure:"retry_delay_multiplier"`
}
