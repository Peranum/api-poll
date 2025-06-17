package config

// Telegram holds the configuration for Telegram integration
type Telegram struct {
	BotID              string `mapstructure:"bot_id"              validate:"required" env:"TELEGRAM_BOT_ID"`
	AuthorizationToken string `mapstructure:"authorization_token" validate:"required" env:"TELEGRAM_AUTHORIZATION_TOKEN"`
	GroupID            int64  `mapstructure:"group_id"            validate:"required" env:"TELEGRAM_GROUP_ID"`
}
