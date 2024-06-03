package config

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	Addr           string    `mapstructure:"addr"`
	Environment    string    `mapstructure:"environment"`
	DSN            string    `mapstructure:"dsn"`
	DebugEnabled   bool      `mapstructure:"debug_enabled"`
	AllowedOrigins []string  `mapstructure:"allowed_origins"`
	Log            logConfig `mapstructure:"log"`
}

type logConfig struct {
	Level         string `mapstructure:"level"`
	HumanReadable bool   `mapstructure:"human_readable"`
}

func GetConfig() Config {
	var cfg Config
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Msg("Error reading config file")
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatal().Err(err).Msg("Unable to decode into struct")
	}

	return cfg
}
