package di

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/config"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/service"
	"github.com/mymmrac/telego"
)

type Container struct {
	Telegram *telego.Bot
	Config   config.Config
	Logger   *slog.Logger
	Metrics  *service.PrometheusService
}

func (c Container) SendMessage(format string, args ...any) {
	now := time.Now().Format("2006-01-02T15:04:05.000Z07:00")

	go func() {
		format = fmt.Sprintf(
			"<b>[upbit.api.poller]</b>\n<u>%s</u>\n\n%s",
			now,
			format,
		)

		body := fmt.Sprintf(format, args...)
		parts := strings.Split(body, "\n")
		builder := strings.Builder{}

		for _, part := range parts {
			if len(part) >= 4096 {
				c.Logger.Error("part is too long", "part", part)
				continue
			}

			if builder.Len()+len(part) >= 4096 {
				if _, err := c.Telegram.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID:    telego.ChatID{ID: c.Config.Telegram.GroupID},
					Text:      builder.String(),
					ParseMode: telego.ModeHTML,
				}); err != nil {
					c.Logger.Error("failed to send telegram message", "error", err)
				}

				builder.Reset()
			}

			builder.WriteString(part)
			builder.WriteString("\n")
		}

		if builder.Len() > 0 {
			if _, err := c.Telegram.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: c.Config.Telegram.GroupID},
				Text:      builder.String(),
				ParseMode: telego.ModeHTML,
			}); err != nil {
				c.Logger.Error("failed to send telegram message", "error", err)
			}
		}
	}()
}
