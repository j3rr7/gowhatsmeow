package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Logger   LoggerConfig   `mapstructure:"logger"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type DatabaseConfig struct {
	Type string `mapstructure:"type"` // e.g. sqlite3 or postgres
	URL  string `mapstructure:"url"`  // e.g. file:whatsapp.db?_foreign_keys=on or postgres://...
}

type LoggerConfig struct {
	Level    string `mapstructure:"level"`    // debug, info, warn, error
	Encoding string `mapstructure:"encoding"` // console or json
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("database.type", "sqlite3")
	viper.SetDefault("database.url", "file:whatsapp.db?_foreign_keys=on")
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.encoding", "console")

	viper.SetEnvPrefix("WA_API")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
