package main

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Debug      bool        `mapstructure:"debug"`
	DB         DBConfig    `mapstructure:"db"`
	Initialise bool        `mapstructure:"initialise"`
	GeoIP      GeoIPConfig `mapstructure:"geoip"`
}

type DBConfig struct {
	Host     string `mapstructure:"host"`
	DB       string `mapstructure:"db"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type GeoIPConfig struct {
	DatabasePath string `mapstructure:"database_path"`
}

func InitConfig() (*Config, error) {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetEnvPrefix("leeio")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	setConfigDefaults()

	config := Config{}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func setConfigDefaults() {
	viper.SetDefault("debug", false)
	viper.SetDefault("db.host", "")
	viper.SetDefault("db.port", 3306)
	viper.SetDefault("db.db", "paste")
	viper.SetDefault("db.user", "")
	viper.SetDefault("db.password", "")
}
