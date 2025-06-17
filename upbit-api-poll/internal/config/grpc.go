package config

import "time"

type GRPC struct {
	Address     string        `mapstructure:"address"      validate:"required" env:"GRPC_ADDRESS"`
	DialTimeout time.Duration `mapstructure:"dial_timeout" validate:"required" env:"GRPC_DIAL_TIMEOUT"`
	CallTimeout time.Duration `mapstructure:"call_timeout" validate:"required" env:"GRPC_CALL_TIMEOUT"`
}
