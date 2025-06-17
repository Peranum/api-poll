package setup

import (
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/config"
)

func mustLogger(cfg config.Config) *slog.Logger {
	logLevel, err := cfg.Logger.ParseSlogLogLevel()
	if err != nil {
		log.Panicf("failed to parse log level: %s", err)
	}

	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: cfg.Logger.AddSource,
	}

	if strings.ToLower(cfg.Logger.Format) == "json" {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}

	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
