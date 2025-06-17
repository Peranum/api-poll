package config

import "time"

type UpbitClient struct {
	UpbitAnnouncementsURL string        `mapstructure:"upbit_announcements_url" validate:"required"      env:"UPBIT_ANNOUNCEMENTS_URL"`
	MaxRetries            int           `mapstructure:"max_retries"             validate:"required,gt=0" env:"MAX_RETRIES"`
	RetryDelay            time.Duration `mapstructure:"retry_delay"             validate:"required,gt=0" env:"RETRY_DELAY"`
	RetryDelayMultiplier  float64       `mapstructure:"retry_delay_multiplier"  validate:"required,gt=0" env:"RETRY_DELAY_MULTIPLIER"`
	MetricsCheckPeriod    time.Duration `mapstructure:"metrics_check_period"    validate:"required,gt=0" env:"METRICS_CHECK_PERIOD"`
}
