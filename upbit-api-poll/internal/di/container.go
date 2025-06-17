package di

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/config"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/service"
	"github.com/mymmrac/telego"
)

type Container struct {
	Telegram *telego.Bot
	Config   config.Config
	Logger   *slog.Logger
	Metrics  service.MetricsService
}

func (c *Container) SendMessage(format string, args ...any) {
	format = fmt.Sprintf(
		"<b>[upbit.api.poller]</b>\n<u>%s</u>\n\n%s",
		time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
		format,
	)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if _, err := c.Telegram.SendMessage(ctx, &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: c.Config.Telegram.GroupID},
			Text:      fmt.Sprintf(format, args...),
			ParseMode: telego.ModeHTML,
		}); err != nil {
			c.Logger.Error("failed to send telegram message", "error", err)
		}
	}()
}
