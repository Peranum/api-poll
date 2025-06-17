package config

import (
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/entity"
)

type APIPoller struct {
	TargetRPS         float64             `mapstructure:"target_rps"           validate:"required,gt=0" env:"TARGET_RPS"`
	SingleProxyMaxRPS float64             `mapstructure:"single_proxy_max_rps" validate:"required,gt=0" env:"SINGLE_PROXY_MAX_RPS"`
	Proxies           []entity.Proxy      `mapstructure:"proxies"              validate:"required,dive" env:"PROXIES"`
	WorkSchedule      entity.WorkSchedule `mapstructure:"work_schedule"        validate:"required"      env:"WORK_SCHEDULE"`
}
