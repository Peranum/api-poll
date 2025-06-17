package setup

import (
	"log"
	"strings"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/config"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/di"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/service"
	"github.com/mymmrac/telego"
)

func MustContainer(cfg config.Config) di.Container {
	logger := mustLogger(cfg)

	token := strings.Join([]string{
		cfg.Telegram.BotID,
		cfg.Telegram.AuthorizationToken,
	}, ":")

	telegram, err := telego.NewBot(token)
	if err != nil {
		log.Panicf("Failed to create telegram bot: %v", err)
	}

	metrics := service.NewPrometheusService()

	return di.Container{
		Config:   cfg,
		Logger:   logger,
		Telegram: telegram,
		Metrics:  metrics,
	}
}
