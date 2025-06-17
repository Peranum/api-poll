package config

import (
	"fmt"
	"log/slog"
	"strings"
)

type Logger struct {
	Level     string `mapstructure:"level"      validate:"required,oneof=debug info warn warning error" env:"LOGGER_LEVEL"`
	Format    string `mapstructure:"format"     validate:"required,oneof=json text"                     env:"LOGGER_FORMAT"`
	AddSource bool   `mapstructure:"add_source"                                                         env:"LOGGER_ADD_SOURCE"`
}

func (l Logger) ParseSlogLogLevel() (slog.Level, error) {
	switch strings.ToLower(l.Level) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level: %s", l.Level)
	}
}
