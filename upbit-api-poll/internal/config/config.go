package config

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/httptools"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	UpbitAPI            UpbitAPI            `mapstructure:"upbit_api"             validate:"required"`
	WebsocketSucker     WebsocketSucker     `mapstructure:"websocket_sucker"      validate:"required"`
	ProxyRotatingPoller ProxyRotatingPoller `mapstructure:"proxy_rotating_poller" validate:"required"`
	Telegram            Telegram            `mapstructure:"telegram"              validate:"required"`
	Logger              Logger              `mapstructure:"logger"                validate:"required"`
	GRPC                GRPC                `mapstructure:"grpc"                  validate:"required"`
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	validate := validator.New()

	return validate.Struct(&c)
}

func (c Config) String() string {
	c.Telegram.AuthorizationToken = "[REDACTED]"
	c.ProxyRotatingPoller.Proxies = []httptools.Proxy{}


	b, _ := json.Marshal(c)

	return strings.ReplaceAll(string(b), `"`, `'`)
}

func MustParseConfig(configPath string) Config {
	_ = godotenv.Load()
	v := viper.New()

	configDir := setConfigName(v, configPath)

	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		log.Printf(
			"Warning reading config file: %s, will use defaults and environment variables\n",
			err,
		)
	}

	v.SetEnvPrefix("UPBITAP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	bindEnvVars(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("Unable to decode config into struct: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	return cfg
}

func setConfigName(v *viper.Viper, configPath string) string {
	configFileInfo, err := filepath.Abs(configPath)
	if err == nil && filepath.Ext(configFileInfo) != "" {
		configFileName := strings.TrimSuffix(
			filepath.Base(configFileInfo),
			filepath.Ext(configFileInfo),
		)
		v.SetConfigName(configFileName)

		return filepath.Dir(configFileInfo)
	}

	v.SetConfigName("config")

	return configPath
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "json")
	v.SetDefault("logger.add_source", false)
	v.SetDefault("grpc.address", "localhost:49999")
	v.SetDefault("grpc.dial_timeout", "5s")
	v.SetDefault("grpc.call_timeout", "10s")
	v.SetDefault("app.metrics_check_period", "5m")
	v.SetDefault("app.single_proxy_max_rps", 0.2)
}

func bindEnvVars(v *viper.Viper) {
	envMappings, err := createEnvMappings(reflect.TypeOf(Config{}))
	if err != nil {
		log.Fatalf("Error creating env mappings: %v", err)
	}

	for configKey, envVar := range envMappings {
		bindEnv(v, configKey, envVar)
	}
}

func createEnvMappings(reflectedType reflect.Type, args ...any) (map[string]string, error) {
	var envMappings map[string]string
	if len(args) > 0 {
		envMappings = args[0].(map[string]string)
	} else {
		envMappings = make(map[string]string)
	}

	var prefixReference string
	if len(args) > 1 {
		prefixReference = args[1].(string)
	}

	for i := 0; i < reflectedType.NumField(); i++ {
		field := reflectedType.Field(i)

		if field.Type.Kind() != reflect.Struct {
			if err := mapStructField(field, envMappings, prefixReference); err != nil {
				return nil, err
			}

			continue
		}

		name := field.Tag.Get("mapstructure")
		if name == "" {
			return nil, fmt.Errorf("mapstructure tag is required for struct field %s", field.Name)
		}

		prefix := name
		if prefixReference != "" {
			prefix = strings.Join([]string{prefixReference, name}, ".")
		}

		if _, err := createEnvMappings(field.Type, envMappings, prefix); err != nil {
			return nil, err
		}
	}

	return envMappings, nil
}

func mapStructField(field reflect.StructField, envMappings map[string]string, prefix string) error {
	mapstructureTag := field.Tag.Get("mapstructure")
	if mapstructureTag == "" {
		return fmt.Errorf("mapstructure tag is required for struct field %s", field.Name)
	}

	envTag := field.Tag.Get("env")
	if envTag == "" || envTag == "-" { // "-" means that the field is not binded to an env var
		return nil
	}

	envMappings[strings.Join([]string{prefix, mapstructureTag}, ".")] = envTag

	return nil
}

func bindEnv(v *viper.Viper, configKey, envVar string) {
	if err := v.BindEnv(configKey, envVar); err != nil {
		log.Printf("Error binding env var %s: %s\n", envVar, err)
	}
}
