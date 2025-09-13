package config

import (
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/entity"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/httptools"
)

type ProxyRotatingPoller struct {
	TargetRPS    float64                       `mapstructure:"target_rps"    validate:"required"      env:"PROXY_ROTATING_POLLER_TARGET_RPS"`
	Proxies      []httptools.Proxy             `mapstructure:"proxies"       validate:"required,dive" env:"PROXY_ROTATING_POLLER_PROXIES"`
	WorkSchedule entity.WorkSchedule           `mapstructure:"work_schedule" validate:"required"      env:"PROXY_ROTATING_POLLER_WORK_SCHEDULE"`
	Retries      httptools.ClientRetriesConfig `mapstructure:"retries"       validate:"required"      env:"PROXY_ROTATING_POLLER_RETRIES"`
}
