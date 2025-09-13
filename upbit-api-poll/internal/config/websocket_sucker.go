package config

type WebsocketSucker struct {
	URL    string `mapstructure:"url"     validate:"required" env:"WEBSOCKET_SUCKER_URL"`
	APIKey string `mapstructure:"api_key" validate:"required" env:"WEBSOCKET_SUCKER_API_KEY"`
}
